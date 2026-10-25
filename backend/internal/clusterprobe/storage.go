// Package clusterprobe implements explicit, bounded operational canaries.
package clusterprobe

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

var ErrProbe = errors.New("storage probe failed; verify endpoint, credentials, permissions and canary receipts")
var identity = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

type Receipt struct {
	Challenge string    `json:"challenge"`
	Node      string    `json:"node"`
	Host      string    `json:"host"`
	Digest    string    `json:"digest"`
	CheckedAt time.Time `json:"checked_at"`
	Read      bool      `json:"read"`
	Scope     string    `json:"scope"`
}
type Client struct {
	s3     *s3.Client
	bucket string
	local  bool
}

func New(endpoint, region, bucket, access, secret string, local bool) (*Client, error) {
	u, e := url.Parse(endpoint)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || bucket == "" || strings.ContainsAny(bucket, "/\\") || access == "" || secret == "" {
		return nil, ErrProbe
	}
	validTransport := u.Scheme == "https" || (local && u.Scheme == "http" && net.ParseIP(u.Hostname()).IsLoopback())
	if !validTransport {
		return nil, ErrProbe
	}
	if region == "" {
		region = "auto"
	}
	// Explicit credentials only; never load a developer's unrelated AWS profile.
	cfg := aws.Config{Region: region, Credentials: credentials.NewStaticCredentialsProvider(access, secret, ""), HTTPClient: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	sdk := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = &endpoint
		o.UsePathStyle = true
		o.RetryMaxAttempts = 1
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
	})
	return &Client{sdk, bucket, local}, nil
}
func key(challenge, node string) (string, error) {
	id, err := uuid.Parse(challenge)
	if err != nil || id.String() != challenge || !identity.MatchString(node) {
		return "", ErrProbe
	}
	return ".sub2api-ha-probe/" + challenge + "/" + node, nil
}
func hostID() string {
	name, _ := os.Hostname()
	sum := sha256.Sum256([]byte(name))
	return hex.EncodeToString(sum[:])
}
func (c *Client) Write(ctx context.Context, challenge, node string) (Receipt, error) {
	r := Receipt{Challenge: challenge, Node: node, Host: hostID(), CheckedAt: time.Now(), Scope: "cross_node_probe"}
	if c.local {
		r.Scope = "local_lab"
	}
	k, err := key(challenge, node)
	if err != nil {
		return r, err
	}
	payload := make([]byte, 1024)
	if _, err = rand.Read(payload); err != nil {
		return r, ErrProbe
	}
	sum := sha256.Sum256(payload)
	r.Digest = hex.EncodeToString(sum[:])
	_, err = c.s3.PutObject(ctx, &s3.PutObjectInput{Bucket: &c.bucket, Key: &k, Body: bytes.NewReader(payload), ContentLength: aws.Int64(int64(len(payload))), ContentType: aws.String("application/octet-stream"), IfNoneMatch: aws.String("*")})
	if err != nil {
		return r, ErrProbe
	}
	return r, nil
}
func (c *Client) Read(ctx context.Context, node string, peer Receipt) (Receipt, error) {
	r := Receipt{Node: node, Host: hostID(), Challenge: peer.Challenge, Digest: peer.Digest, CheckedAt: time.Now(), Read: true, Scope: peer.Scope}
	k, err := key(peer.Challenge, peer.Node)
	if err != nil || !identity.MatchString(node) || node == peer.Node || peer.Read || len(peer.Digest) != 64 || time.Since(peer.CheckedAt) > time.Hour || time.Until(peer.CheckedAt) > 5*time.Second {
		return r, ErrProbe
	}
	if (peer.Scope != "local_lab" && peer.Scope != "cross_node_probe") || c.local != (peer.Scope == "local_lab") || (!c.local && peer.Host == r.Host) {
		return r, ErrProbe
	}
	object, err := c.s3.GetObject(ctx, &s3.GetObjectInput{Bucket: &c.bucket, Key: &k})
	if err != nil {
		return r, ErrProbe
	}
	defer func() { _ = object.Body.Close() }()
	payload, err := io.ReadAll(io.LimitReader(object.Body, 1025))
	if err != nil || len(payload) != 1024 {
		return r, ErrProbe
	}
	sum := sha256.Sum256(payload)
	if hex.EncodeToString(sum[:]) != peer.Digest {
		return r, ErrProbe
	}
	return r, nil
}
func (c *Client) Cleanup(ctx context.Context, receipt Receipt) error {
	k, err := key(receipt.Challenge, receipt.Node)
	if err != nil || receipt.Read || receipt.Host != hostID() {
		return ErrProbe
	}
	// Only the exact canary owned by this host, never list or purge a bucket.
	_, err = c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &c.bucket, Key: &k})
	if err != nil {
		return ErrProbe
	}
	return nil
}
