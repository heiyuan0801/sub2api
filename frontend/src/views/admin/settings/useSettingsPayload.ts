import {
  appendAuthSourceDefaultsToUpdateRequest,
  defaultWeChatConnectScopesForMode,
  sanitizeAccountSchedulingThresholdsMap,
  sanitizePlatformQuotasMap,
  type AccountSchedulingThresholdsMap,
  type AuthSourceDefaultsState,
  type DefaultPlatformQuotasMap,
  type DefaultSubscriptionSetting,
  type OpenAIFastPolicyRule,
  type SystemSettings,
  type UpdateSettingsRequest,
  type WeChatConnectMode,
} from "@/api/admin/settings";
import {
  sanitizeRechargeBonusTiersForSubmit,
  type RechargeBonusMode,
  type RechargeBonusTierDraft,
} from "@/utils/rechargeBonus";
import {
  serializeFingerprintRowsToJSON,
  type FingerprintSignalRow,
} from "../codexFingerprintSignals";

export type SettingsForm = Omit<
  SystemSettings,
  | "wechat_connect_open_enabled"
  | "wechat_connect_mp_enabled"
  | "wechat_connect_mobile_enabled"
  | "openai_oauth_scheduling_rate_multiplier"
> & {
  /** Form always binds a concrete boolean (SystemSettings marks this optional). */
  channel_monitor_hide_throughput: boolean;
  channel_monitor_show_quota: boolean;
  channel_monitor_hide_user_ranking: boolean;
  smtp_password: string;
  turnstile_secret_key: string;
  tencent_captcha_app_secret_key: string;
  tencent_captcha_cloud_secret_id: string;
  tencent_captcha_cloud_secret_key: string;
  aliyun_captcha_access_key_secret: string;
  linuxdo_connect_client_secret: string;
  dingtalk_connect_client_secret: string;
  wechat_connect_app_secret: string;
  wechat_connect_open_app_secret: string;
  wechat_connect_mp_app_secret: string;
  wechat_connect_mobile_app_secret: string;
  wechat_connect_open_enabled: boolean;
  wechat_connect_mp_enabled: boolean;
  wechat_connect_mobile_enabled: boolean;
  oidc_connect_client_secret: string;
  github_oauth_client_secret: string;
  google_oauth_client_secret: string;
  force_email_on_third_party_signup: boolean;
  openai_low_upstream_rate_priority_enabled: boolean;
  openai_oauth_scheduling_rate_multiplier: number | "" | null;
  openai_advanced_scheduler_enabled: boolean;
  openai_advanced_scheduler_sticky_weighted_enabled: boolean;
  openai_advanced_scheduler_subscription_priority_enabled: boolean;
  openai_advanced_scheduler_lb_top_k: string;
  openai_advanced_scheduler_weight_priority: string;
  openai_advanced_scheduler_weight_load: string;
  openai_advanced_scheduler_weight_queue: string;
  openai_advanced_scheduler_weight_error_rate: string;
  openai_advanced_scheduler_weight_ttft: string;
  openai_advanced_scheduler_weight_reset: string;
  openai_advanced_scheduler_weight_quota_headroom: string;
  openai_advanced_scheduler_weight_upstream_cost: string;
  openai_advanced_scheduler_weight_previous_response: string;
  openai_advanced_scheduler_weight_session_sticky: string;
  // 充值赠送阶梯编辑态：允许留空的行，提交时清洗为 RechargeBonusTier[]
  payment_recharge_bonus_tiers: RechargeBonusTierDraft[];
  payment_recharge_bonus_mode: RechargeBonusMode;
  payment_recharge_bonus_notice: string;
  // 系统全局平台限额 map；form 内始终归一化为全部平台对象（模板非空绑定依赖此不变量）
  default_platform_quotas: DefaultPlatformQuotasMap;
  account_scheduling_thresholds: AccountSchedulingThresholdsMap;
};

export interface CodexClientRow {
  originator: string;
  uaContains: string; // 逗号分隔，序列化时拆成 ua_contains 数组
  skipEngineFingerprint?: boolean; // 仅白名单：命中即跳过引擎指纹门
}
function serializeCodexRowsToJSON(rows: CodexClientRow[]): string {
  const entries = rows
    .map((r) => {
      const entry: {
        originator: string;
        ua_contains: string[];
        skip_engine_fingerprint?: boolean;
      } = {
        originator: r.originator.trim(),
        ua_contains: r.uaContains
          .split(",")
          .map((s) => s.trim())
          .filter((s) => s.length > 0),
      };
      if (r.skipEngineFingerprint) entry.skip_engine_fingerprint = true;
      return entry;
    })
    .filter((e) => e.originator !== "" || e.ua_contains.length > 0);
  return entries.length > 0 ? JSON.stringify(entries) : "";
}

export function isValidOptionalSettingsHttpUrl(url: string): boolean {
  if (!url) return true;
  try {
    const u = new URL(url);
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
}

interface SettingsPayloadContext {
  form: SettingsForm;
  registrationEmailSuffixWhitelistTags: string[];
  normalizedDefaultSubscriptions: DefaultSubscriptionSetting[];
  wechatStoredMode: WeChatConnectMode;
  claudeOAuthSystemPromptBlocksJSON: string;
  oauthSchedulingRate: SettingsForm["openai_oauth_scheduling_rate_multiplier"];
  codexFingerprintRows: FingerprintSignalRow[];
  codexBlacklistRows: CodexClientRow[];
  codexWhitelistRows: CodexClientRow[];
  currentOrigin: string;
  openaiFastPolicyLoaded: boolean;
  openaiFastPolicyRules: OpenAIFastPolicyRule[];
  authSourceDefaults: AuthSourceDefaultsState;
}

/**
 * Assemble the validated settings draft for the existing update endpoint.
 * The legacy origin fallback is also reflected in the recharge URL input.
 */
export function buildSettingsPayload({
  form,
  registrationEmailSuffixWhitelistTags,
  normalizedDefaultSubscriptions,
  wechatStoredMode,
  claudeOAuthSystemPromptBlocksJSON,
  oauthSchedulingRate,
  codexFingerprintRows,
  codexBlacklistRows,
  codexWhitelistRows,
  currentOrigin,
  openaiFastPolicyLoaded,
  openaiFastPolicyRules,
  authSourceDefaults,
}: SettingsPayloadContext): UpdateSettingsRequest {
  const payload: UpdateSettingsRequest = {
    registration_enabled: form.registration_enabled,
    email_verify_enabled: form.email_verify_enabled,
    registration_email_suffix_whitelist:
      registrationEmailSuffixWhitelistTags.map((suffix) =>
        suffix.startsWith("*.") ? suffix : `@${suffix}`,
      ),
    registration_email_domain_quota_enabled:
      form.registration_email_domain_quota_enabled,
    promo_code_enabled: form.promo_code_enabled,
    invitation_code_enabled: form.invitation_code_enabled,
    password_reset_enabled: form.password_reset_enabled,
    totp_enabled: form.totp_enabled,
    passkey_enabled: form.passkey_enabled,
    session_binding_enabled: form.session_binding_enabled,
    step_up_enabled: form.step_up_enabled,
    // 清空数字框时 v-model.number 会得到空串，后端 int 字段解析空串会 400 拒绝整次保存；
    // 空/非法值回退默认 180（与后端 parseAuditLogRetentionDays("") 语义一致，0 仍表示永久保留）。
    audit_log_retention_days: Number.isFinite(form.audit_log_retention_days)
      ? form.audit_log_retention_days
      : 180,
    login_agreement_enabled: form.login_agreement_enabled,
    login_agreement_mode: form.login_agreement_mode,
    login_agreement_updated_at: form.login_agreement_updated_at,
    login_agreement_documents: form.login_agreement_documents,
    default_balance: form.default_balance,
    affiliate_rebate_rate: Math.min(
      100,
      Math.max(0, Number(form.affiliate_rebate_rate) || 0),
    ),
    affiliate_rebate_freeze_hours: Math.max(0, Math.min(720, Number(form.affiliate_rebate_freeze_hours) || 0)),
    affiliate_rebate_duration_days: Math.max(0, Math.min(3650, Math.floor(Number(form.affiliate_rebate_duration_days) || 0))),
    affiliate_rebate_per_invitee_cap: Math.max(0, Number(form.affiliate_rebate_per_invitee_cap) || 0),
    affiliate_admin_recharge_enabled: form.affiliate_admin_recharge_enabled,
    default_concurrency: form.default_concurrency,
    default_subscriptions: normalizedDefaultSubscriptions,
    force_email_on_third_party_signup: form.force_email_on_third_party_signup,
    default_user_rpm_limit: form.default_user_rpm_limit,
    site_name: form.site_name,
    site_logo: form.site_logo,
    site_subtitle: form.site_subtitle,
    api_base_url: form.api_base_url,
    contact_info: form.contact_info,
    doc_url: form.doc_url,
    home_content: form.home_content,
    compact_home_enabled: form.compact_home_enabled,
    backend_mode_enabled: form.backend_mode_enabled,
    hide_ccs_import_button: form.hide_ccs_import_button,
    table_default_page_size: form.table_default_page_size,
    table_page_size_options: form.table_page_size_options,
    custom_menu_items: form.custom_menu_items,
    custom_endpoints: form.custom_endpoints,
    frontend_url: form.frontend_url,
    smtp_host: form.smtp_host,
    smtp_port: form.smtp_port,
    smtp_username: form.smtp_username,
    smtp_password: form.smtp_password || undefined,
    smtp_from_email: form.smtp_from_email,
    smtp_from_name: form.smtp_from_name,
    smtp_use_tls: form.smtp_use_tls,
    turnstile_enabled: form.turnstile_enabled,
    turnstile_site_key: form.turnstile_site_key,
    turnstile_secret_key: form.turnstile_secret_key || undefined,
    tencent_captcha_enabled: form.tencent_captcha_enabled,
    tencent_captcha_app_id: form.tencent_captcha_app_id,
    tencent_captcha_app_secret_key:
      form.tencent_captcha_app_secret_key || undefined,
    tencent_captcha_cloud_secret_id:
      form.tencent_captcha_cloud_secret_id || undefined,
    tencent_captcha_cloud_secret_key:
      form.tencent_captcha_cloud_secret_key || undefined,
    tencent_captcha_region: form.tencent_captcha_region,
    aliyun_captcha_enabled: form.aliyun_captcha_enabled,
    aliyun_captcha_access_key_id: form.aliyun_captcha_access_key_id,
    aliyun_captcha_access_key_secret:
      form.aliyun_captcha_access_key_secret || undefined,
    aliyun_captcha_scene_id: form.aliyun_captcha_scene_id,
    aliyun_captcha_prefix: form.aliyun_captcha_prefix,
    aliyun_captcha_region: form.aliyun_captcha_region,
    api_key_acl_trust_forwarded_ip: form.api_key_acl_trust_forwarded_ip,
    forwarded_client_ip_headers: form.forwarded_client_ip_headers,
    linuxdo_connect_enabled: form.linuxdo_connect_enabled,
    linuxdo_connect_client_id: form.linuxdo_connect_client_id,
    linuxdo_connect_client_secret:
      form.linuxdo_connect_client_secret || undefined,
    linuxdo_connect_redirect_url: form.linuxdo_connect_redirect_url,
    dingtalk_connect_enabled: form.dingtalk_connect_enabled,
    dingtalk_connect_client_id: form.dingtalk_connect_client_id,
    dingtalk_connect_client_secret:
      form.dingtalk_connect_client_secret || undefined,
    dingtalk_connect_redirect_url: form.dingtalk_connect_redirect_url,
    dingtalk_connect_corp_restriction_policy:
      form.dingtalk_connect_corp_restriction_policy,
    dingtalk_connect_internal_corp_id: form.dingtalk_connect_internal_corp_id,
    dingtalk_connect_bypass_registration: form.dingtalk_connect_bypass_registration,
    dingtalk_connect_sync_corp_email: form.dingtalk_connect_sync_corp_email,
    dingtalk_connect_sync_display_name: form.dingtalk_connect_sync_display_name,
    dingtalk_connect_sync_dept: form.dingtalk_connect_sync_dept,
    dingtalk_connect_sync_corp_email_attr_key: form.dingtalk_connect_sync_corp_email_attr_key,
    dingtalk_connect_sync_display_name_attr_key: form.dingtalk_connect_sync_display_name_attr_key,
    dingtalk_connect_sync_dept_attr_key: form.dingtalk_connect_sync_dept_attr_key,
    dingtalk_connect_sync_corp_email_attr_name: form.dingtalk_connect_sync_corp_email_attr_name,
    dingtalk_connect_sync_display_name_attr_name: form.dingtalk_connect_sync_display_name_attr_name,
    dingtalk_connect_sync_dept_attr_name: form.dingtalk_connect_sync_dept_attr_name,
    wechat_connect_enabled: form.wechat_connect_enabled,
    wechat_connect_app_id:
      form.wechat_connect_open_app_id ||
      form.wechat_connect_mp_app_id ||
      form.wechat_connect_mobile_app_id ||
      form.wechat_connect_app_id,
    wechat_connect_app_secret: form.wechat_connect_app_secret || undefined,
    wechat_connect_open_app_id: form.wechat_connect_open_app_id,
    wechat_connect_open_app_secret:
      form.wechat_connect_open_app_secret || undefined,
    wechat_connect_mp_app_id: form.wechat_connect_mp_app_id,
    wechat_connect_mp_app_secret:
      form.wechat_connect_mp_app_secret || undefined,
    wechat_connect_mobile_app_id: form.wechat_connect_mobile_app_id,
    wechat_connect_mobile_app_secret:
      form.wechat_connect_mobile_app_secret || undefined,
    wechat_connect_open_enabled: form.wechat_connect_open_enabled,
    wechat_connect_mp_enabled: form.wechat_connect_mp_enabled,
    wechat_connect_mobile_enabled: form.wechat_connect_mobile_enabled,
    wechat_connect_mode: wechatStoredMode,
    wechat_connect_scopes:
      defaultWeChatConnectScopesForMode(wechatStoredMode),
    wechat_connect_redirect_url: form.wechat_connect_redirect_url,
    wechat_connect_frontend_redirect_url:
      form.wechat_connect_frontend_redirect_url,
    oidc_connect_enabled: form.oidc_connect_enabled,
    oidc_connect_provider_name: form.oidc_connect_provider_name,
    oidc_connect_client_id: form.oidc_connect_client_id,
    oidc_connect_client_secret: form.oidc_connect_client_secret || undefined,
    oidc_connect_issuer_url: form.oidc_connect_issuer_url,
    oidc_connect_discovery_url: form.oidc_connect_discovery_url,
    oidc_connect_authorize_url: form.oidc_connect_authorize_url,
    oidc_connect_token_url: form.oidc_connect_token_url,
    oidc_connect_userinfo_url: form.oidc_connect_userinfo_url,
    oidc_connect_jwks_url: form.oidc_connect_jwks_url,
    oidc_connect_scopes: form.oidc_connect_scopes,
    oidc_connect_redirect_url: form.oidc_connect_redirect_url,
    oidc_connect_frontend_redirect_url:
      form.oidc_connect_frontend_redirect_url,
    oidc_connect_token_auth_method: form.oidc_connect_token_auth_method,
    oidc_connect_use_pkce: form.oidc_connect_use_pkce,
    oidc_connect_validate_id_token: form.oidc_connect_validate_id_token,
    oidc_connect_allowed_signing_algs: form.oidc_connect_allowed_signing_algs,
    oidc_connect_clock_skew_seconds: form.oidc_connect_clock_skew_seconds,
    oidc_connect_require_email_verified:
      form.oidc_connect_require_email_verified,
    oidc_connect_userinfo_email_path: form.oidc_connect_userinfo_email_path,
    oidc_connect_userinfo_id_path: form.oidc_connect_userinfo_id_path,
    oidc_connect_userinfo_username_path:
      form.oidc_connect_userinfo_username_path,
    github_oauth_enabled: form.github_oauth_enabled,
    github_oauth_client_id: form.github_oauth_client_id,
    github_oauth_client_secret:
      form.github_oauth_client_secret || undefined,
    github_oauth_redirect_url: form.github_oauth_redirect_url,
    github_oauth_frontend_redirect_url:
      form.github_oauth_frontend_redirect_url,
    google_oauth_enabled: form.google_oauth_enabled,
    google_oauth_client_id: form.google_oauth_client_id,
    google_oauth_client_secret:
      form.google_oauth_client_secret || undefined,
    google_oauth_redirect_url: form.google_oauth_redirect_url,
    google_oauth_frontend_redirect_url:
      form.google_oauth_frontend_redirect_url,
    enable_model_fallback: form.enable_model_fallback,
    fallback_model_anthropic: form.fallback_model_anthropic,
    fallback_model_openai: form.fallback_model_openai,
    fallback_model_gemini: form.fallback_model_gemini,
    fallback_model_antigravity: form.fallback_model_antigravity,
    grok_default_text_model:
      form.grok_default_text_model.trim() || "grok-4.5",
    grok_cross_client_model_map_enabled:
      form.grok_cross_client_model_map_enabled,
    grok_default_base_url_mode: form.grok_default_base_url_mode,
    enable_identity_patch: form.enable_identity_patch,
    identity_patch_prompt: form.identity_patch_prompt,
    min_claude_code_version: form.min_claude_code_version,
    max_claude_code_version: form.max_claude_code_version,
    allow_ungrouped_key_scheduling: form.allow_ungrouped_key_scheduling,
    openai_ttft_mode:
      form.openai_ttft_mode === "visible" ? "visible" : "semantic",
    enable_fingerprint_unification: form.enable_fingerprint_unification,
    enable_metadata_passthrough: form.enable_metadata_passthrough,
    enable_cch_signing: form.enable_cch_signing,
    enable_claude_oauth_system_prompt_injection:
      form.enable_claude_oauth_system_prompt_injection,
    claude_oauth_system_prompt: form.claude_oauth_system_prompt?.trim()
      ? form.claude_oauth_system_prompt
      : "",
    claude_oauth_system_prompt_blocks: claudeOAuthSystemPromptBlocksJSON,
    enable_anthropic_cache_ttl_1h_injection:
      form.enable_anthropic_cache_ttl_1h_injection,
    rewrite_message_cache_control: form.rewrite_message_cache_control,
    enable_client_dateline_normalization:
      form.enable_client_dateline_normalization,
    antigravity_user_agent_version:
      form.antigravity_user_agent_version?.trim() || "",
    openai_codex_user_agent:
      form.openai_codex_user_agent?.trim() || "",
    openai_codex_client_version:
      form.openai_codex_client_version?.trim() || "",
    openai_codex_version_auto_sync_enabled:
      form.openai_codex_version_auto_sync_enabled,
    claude_code_client_version: form.claude_code_client_version?.trim() || "",
    claude_code_version_auto_sync_enabled:
      form.claude_code_version_auto_sync_enabled,
    min_codex_version: form.min_codex_version?.trim() || "",
    max_codex_version: form.max_codex_version?.trim() || "",
    codex_cli_only_allow_app_server_clients:
      form.codex_cli_only_allow_app_server_clients,
    codex_cli_only_engine_fingerprint_signals: serializeFingerprintRowsToJSON(
      codexFingerprintRows,
    ),
    codex_cli_only_blacklist: serializeCodexRowsToJSON(
      codexBlacklistRows,
    ),
    codex_cli_only_whitelist: serializeCodexRowsToJSON(
      codexWhitelistRows,
    ),
    // Payment configuration
    payment_enabled: form.payment_enabled,
    risk_control_enabled: form.risk_control_enabled,
    cyber_policy_user_allowlist: form.cyber_policy_user_allowlist,
    cyber_session_block_enabled: form.cyber_session_block_enabled,
    cyber_session_block_ttl_seconds:
      Number(form.cyber_session_block_ttl_seconds) || 3600,
    payment_min_amount: Number(form.payment_min_amount) || 0,
    payment_max_amount: Number(form.payment_max_amount) || 0,
    payment_daily_limit: Number(form.payment_daily_limit) || 0,
    payment_max_pending_orders: Number(form.payment_max_pending_orders) || 0,
    payment_order_timeout_minutes:
      Number(form.payment_order_timeout_minutes) || 0,
    payment_balance_disabled: form.payment_balance_disabled,
    payment_balance_recharge_multiplier:
      Number(form.payment_balance_recharge_multiplier) || 1,
    payment_subscription_usd_to_cny_rate:
      Number(form.payment_subscription_usd_to_cny_rate) || 0,
    payment_recharge_fee_rate: Number(form.payment_recharge_fee_rate) || 0,
    payment_recharge_bonus_tiers: sanitizeRechargeBonusTiersForSubmit(
      form.payment_recharge_bonus_tiers,
    ),
    payment_recharge_bonus_mode: form.payment_recharge_bonus_mode,
    payment_recharge_bonus_notice: form.payment_recharge_bonus_notice,
    payment_enabled_types: form.payment_enabled_types,
    payment_load_balance_strategy: form.payment_load_balance_strategy,
    payment_product_name_prefix: form.payment_product_name_prefix,
    payment_product_name_suffix: form.payment_product_name_suffix,
    payment_help_image_url: form.payment_help_image_url,
    payment_help_text: form.payment_help_text,
    payment_cancel_rate_limit_enabled: form.payment_cancel_rate_limit_enabled,
    payment_cancel_rate_limit_max:
      Number(form.payment_cancel_rate_limit_max) || 10,
    payment_cancel_rate_limit_window:
      Number(form.payment_cancel_rate_limit_window) || 1,
    payment_cancel_rate_limit_unit: form.payment_cancel_rate_limit_unit,
    payment_cancel_rate_limit_window_mode:
      form.payment_cancel_rate_limit_window_mode,
    payment_alipay_force_qrcode: form.payment_alipay_force_qrcode,
    payment_alipay_mobile_precreate_deep_link:
      form.payment_alipay_mobile_precreate_deep_link,
    openai_low_upstream_rate_priority_enabled:
      form.openai_low_upstream_rate_priority_enabled,
    openai_oauth_scheduling_rate_multiplier:
      oauthSchedulingRate === "" ? null : oauthSchedulingRate,
    openai_advanced_scheduler_enabled: form.openai_advanced_scheduler_enabled,
    openai_advanced_scheduler_sticky_weighted_enabled:
      form.openai_advanced_scheduler_sticky_weighted_enabled,
    openai_advanced_scheduler_subscription_priority_enabled:
      form.openai_advanced_scheduler_subscription_priority_enabled,
    openai_advanced_scheduler_lb_top_k:
      form.openai_advanced_scheduler_lb_top_k.trim(),
    openai_advanced_scheduler_weight_priority:
      form.openai_advanced_scheduler_weight_priority.trim(),
    openai_advanced_scheduler_weight_load:
      form.openai_advanced_scheduler_weight_load.trim(),
    openai_advanced_scheduler_weight_queue:
      form.openai_advanced_scheduler_weight_queue.trim(),
    openai_advanced_scheduler_weight_error_rate:
      form.openai_advanced_scheduler_weight_error_rate.trim(),
    openai_advanced_scheduler_weight_ttft:
      form.openai_advanced_scheduler_weight_ttft.trim(),
    openai_advanced_scheduler_weight_reset:
      form.openai_advanced_scheduler_weight_reset.trim(),
    openai_advanced_scheduler_weight_quota_headroom:
      form.openai_advanced_scheduler_weight_quota_headroom.trim(),
    openai_advanced_scheduler_weight_upstream_cost:
      form.openai_advanced_scheduler_weight_upstream_cost.trim(),
    openai_advanced_scheduler_weight_previous_response:
      form.openai_advanced_scheduler_weight_previous_response.trim(),
    openai_advanced_scheduler_weight_session_sticky:
      form.openai_advanced_scheduler_weight_session_sticky.trim(),
    // 余额、订阅到期与账号限额通知
    balance_low_notify_enabled: form.balance_low_notify_enabled,
    balance_low_notify_threshold:
      Number(form.balance_low_notify_threshold) || 0,
    balance_low_notify_recharge_url: (form.balance_low_notify_recharge_url =
      form.balance_low_notify_recharge_url || currentOrigin),
    subscription_expiry_notify_enabled:
      form.subscription_expiry_notify_enabled,
    account_quota_notify_enabled: form.account_quota_notify_enabled,
    account_quota_notify_emails: (
      form.account_quota_notify_emails || []
    ).filter((e) => e.email.trim() !== ""),
    // Channel Monitor feature switch
    channel_monitor_enabled: form.channel_monitor_enabled,
    channel_monitor_mode: form.channel_monitor_mode === 'v1' ? 'v1' : 'v2',
    channel_monitor_default_interval_seconds:
      Number(form.channel_monitor_default_interval_seconds) || 60,
    channel_monitor_hide_throughput: Boolean(form.channel_monitor_hide_throughput),
    channel_monitor_show_quota: Boolean(form.channel_monitor_show_quota),
    channel_monitor_hide_user_ranking: Boolean(form.channel_monitor_hide_user_ranking),
    // Available Channels feature switch
    available_channels_enabled: form.available_channels_enabled,
    // Subscription feature switch
    subscription_enabled: form.subscription_enabled,
    // Model Plaza feature switches + description
    model_plaza_enabled: form.model_plaza_enabled,
    model_plaza_require_auth: form.model_plaza_require_auth,
    model_plaza_description: form.model_plaza_description,
    plugin_management_enabled: form.plugin_management_enabled,
    // Affiliate (邀请返利) feature switch
    affiliate_enabled: form.affiliate_enabled,
    allow_user_view_error_requests: form.allow_user_view_error_requests,
  };

  // 仅当 openai_fast_policy_settings 已成功从后端加载时才回写，
  // 否则省略整个字段，让后端保留既有规则（含默认值）。
  if (openaiFastPolicyLoaded) {
    payload.openai_fast_policy_settings = {
      rules: openaiFastPolicyRules.map((rule) => {
        const whitelist = (rule.model_whitelist || [])
          .map((p) => p.trim())
          .filter((p) => p !== "");
        const hasWhitelist = whitelist.length > 0;
        return {
          service_tier: rule.service_tier,
          action: rule.action,
          scope: rule.scope,
          user_ids:
            rule.user_ids && rule.user_ids.length > 0
              ? [...rule.user_ids]
              : undefined,
          error_message:
            rule.action === "block" ? rule.error_message : undefined,
          model_whitelist: hasWhitelist ? whitelist : undefined,
          fallback_action: hasWhitelist
            ? rule.fallback_action || "pass"
            : undefined,
          fallback_error_message:
            hasWhitelist && rule.fallback_action === "block"
              ? rule.fallback_error_message
              : undefined,
        };
      }),
    };
  }

  payload.default_platform_quotas = sanitizePlatformQuotasMap(form.default_platform_quotas);
  payload.account_scheduling_thresholds = sanitizeAccountSchedulingThresholdsMap(
    form.account_scheduling_thresholds,
  );
  appendAuthSourceDefaultsToUpdateRequest(payload, authSourceDefaults);
  return payload;
}
