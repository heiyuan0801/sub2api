import { describe, expect, it } from "vitest";
import { buildAuthSourceDefaultsState } from "@/api/admin/settings";
import {
  buildSettingsPayload,
  isValidOptionalSettingsHttpUrl,
  type SettingsForm,
} from "../useSettingsPayload";

function legacyDraft() {
  // These fields exercise the pre-split payload’s transformations. Unrelated
  // pass-through fields may be omitted in this fixture; the view tests cover
  // the complete form and update endpoint.
  const values = {
    site_name: "Example Site",
    table_default_page_size: 25,
    table_page_size_options: [10, 25, 50],
    audit_log_retention_days: Number.NaN,
    affiliate_rebate_rate: 120,
    affiliate_rebate_freeze_hours: 900,
    affiliate_rebate_duration_days: 2.9,
    affiliate_rebate_per_invitee_cap: -3,
    smtp_password: "",
    github_oauth_client_secret: "",
    wechat_connect_open_app_id: "fixture-open",
    wechat_connect_mp_app_id: "fixture-mp",
    grok_default_text_model: " ",
    openai_ttft_mode: "visible",
    openai_advanced_scheduler_lb_top_k: " 10 ",
    openai_advanced_scheduler_weight_priority: " 2 ",
    openai_advanced_scheduler_weight_load: "",
    openai_advanced_scheduler_weight_queue: "",
    openai_advanced_scheduler_weight_error_rate: "",
    openai_advanced_scheduler_weight_ttft: "",
    openai_advanced_scheduler_weight_reset: "",
    openai_advanced_scheduler_weight_quota_headroom: "",
    openai_advanced_scheduler_weight_upstream_cost: "",
    openai_advanced_scheduler_weight_previous_response: "",
    openai_advanced_scheduler_weight_session_sticky: "",
    balance_low_notify_recharge_url: "",
    account_quota_notify_emails: [],
    payment_recharge_bonus_tiers: [
      { min_amount: Number.NaN, bonus_percent: 5 },
      { min_amount: 10, bonus_percent: 8 },
    ],
    default_platform_quotas: { openai: { daily: 0, weekly: -2, monthly: 3.5 } },
    account_scheduling_thresholds: {
      openai: 120,
      anthropic: 60.9,
      grok: 0,
      kimi: 100,
      zhipu: 100,
      minimax: 100,
      opencode_go: 100,
      command_code: 100,
    },
  } satisfies Partial<SettingsForm>;

  return values as SettingsForm;
}

function legacyContext(): Parameters<typeof buildSettingsPayload>[0] {
  const authSourceDefaults = buildAuthSourceDefaultsState({});
  authSourceDefaults.email = {
    balance: 1.5,
    concurrency: 3.7,
    subscriptions: [{ group_id: 9, validity_days: 40000 }],
    grant_on_signup: true,
    grant_on_first_bind: false,
    platform_quotas: { openai: { daily: 0, weekly: -2, monthly: null } },
  };

  return {
    form: legacyDraft(),
    registrationEmailSuffixWhitelistTags: ["example.com", "*.example.org"],
    normalizedDefaultSubscriptions: [{ group_id: 4, validity_days: 30 }],
    wechatStoredMode: "mp",
    claudeOAuthSystemPromptBlocksJSON: "[]",
    oauthSchedulingRate: "",
    codexFingerprintRows: [
      { type: "header_exact", match: " x-origin / / x-source ", required: true },
    ],
    codexBlacklistRows: [],
    codexWhitelistRows: [
      { originator: " desktop ", uaContains: " app , , viewer ", skipEngineFingerprint: true },
      { originator: " ", uaContains: "" },
    ],
    currentOrigin: "https://console.example.com",
    openaiFastPolicyLoaded: false,
    openaiFastPolicyRules: [],
    authSourceDefaults,
  };
}

describe("settings payload parity", () => {
  it("keeps the previous wire values, clearing sentinels, nested quotas and input fallback", () => {
    const context = legacyContext();
    const payload = buildSettingsPayload(context);
    // Compare as sent over JSON, where undefined means leave the saved value unchanged.
    const wire = JSON.parse(JSON.stringify(payload));

    expect(wire).toMatchObject({
      site_name: "Example Site",
      table_default_page_size: 25,
      table_page_size_options: [10, 25, 50],
      audit_log_retention_days: 180,
      affiliate_rebate_rate: 100,
      affiliate_rebate_freeze_hours: 720,
      affiliate_rebate_duration_days: 2,
      affiliate_rebate_per_invitee_cap: 0,
      registration_email_suffix_whitelist: ["@example.com", "*.example.org"],
      default_subscriptions: [{ group_id: 4, validity_days: 30 }],
      wechat_connect_app_id: "fixture-open",
      wechat_connect_mode: "mp",
      wechat_connect_scopes: "snsapi_userinfo",
      grok_default_text_model: "grok-4.5",
      openai_ttft_mode: "visible",
      openai_oauth_scheduling_rate_multiplier: null,
      openai_advanced_scheduler_lb_top_k: "10",
      openai_advanced_scheduler_weight_priority: "2",
      balance_low_notify_recharge_url: "https://console.example.com",
      claude_oauth_system_prompt_blocks: "[]",
      codex_cli_only_engine_fingerprint_signals: '[{"type":"header_exact","match":["x-origin","x-source"],"required":true}]',
      codex_cli_only_blacklist: "",
      codex_cli_only_whitelist: '[{"originator":"desktop","ua_contains":["app","viewer"],"skip_engine_fingerprint":true}]',
      payment_recharge_bonus_tiers: [{ min_amount: 10, bonus_percent: 8 }],
      default_platform_quotas: { openai: { daily: 0, weekly: null, monthly: 3.5 } },
      account_scheduling_thresholds: { openai: 100, anthropic: 60, grok: 1 },
      auth_source_default_email_balance: 1.5,
      auth_source_default_email_concurrency: 3,
      auth_source_default_email_subscriptions: [{ group_id: 9, validity_days: 36500 }],
      auth_source_default_email_grant_on_signup: true,
      auth_source_default_email_grant_on_first_bind: false,
      auth_source_default_email_platform_quotas: { openai: { daily: 0, weekly: null, monthly: null } },
    });
    expect(wire).not.toHaveProperty("smtp_password");
    expect(wire).not.toHaveProperty("github_oauth_client_secret");
    expect(wire).not.toHaveProperty("openai_fast_policy_settings");
    expect(context.form.balance_low_notify_recharge_url).toBe("https://console.example.com");
  });

  it("keeps loaded policy rules, skips absent optional fields, and accepts explicit zero", () => {
    const context = legacyContext();
    context.form.smtp_password = "fixture-value";
    context.form.balance_low_notify_recharge_url = "https://billing.example.com";
    context.oauthSchedulingRate = 0;
    context.openaiFastPolicyLoaded = true;
    context.openaiFastPolicyRules = [
      {
        service_tier: "priority",
        action: "block",
        scope: "oauth",
        user_ids: [12],
        error_message: "Not available",
        model_whitelist: [" selected-model ", " "],
        fallback_action: "block",
        fallback_error_message: "Fallback unavailable",
      },
      {
        service_tier: "flex",
        action: "pass",
        scope: "all",
        user_ids: [],
        error_message: "Unused",
        model_whitelist: [" "],
        fallback_action: "block",
        fallback_error_message: "Unused",
      },
    ];
    const payload = buildSettingsPayload(context);
    const wire = JSON.parse(JSON.stringify(payload));

    expect(wire.openai_fast_policy_settings).toEqual({
      rules: [
        {
          service_tier: "priority",
          action: "block",
          scope: "oauth",
          user_ids: [12],
          error_message: "Not available",
          model_whitelist: ["selected-model"],
          fallback_action: "block",
          fallback_error_message: "Fallback unavailable",
        },
        { service_tier: "flex", action: "pass", scope: "all" },
      ],
    });
    expect(payload.openai_fast_policy_settings?.rules[0]?.user_ids)
      .not.toBe(context.openaiFastPolicyRules[0]?.user_ids);
    expect(wire.smtp_password).toBe("fixture-value");
    expect(wire.openai_oauth_scheduling_rate_multiplier).toBe(0);
    expect(wire.balance_low_notify_recharge_url).toBe("https://billing.example.com");
    expect(buildSettingsPayload({ ...context, openaiFastPolicyRules: [] }).openai_fast_policy_settings)
      .toEqual({ rules: [] });
  });
});

describe("optional settings URLs", () => {
  it("retains the old blank and absolute HTTP(S) acceptance rules", () => {
    for (const url of ["", "https://example.com/docs", "http://example.com"]) {
      expect(isValidOptionalSettingsHttpUrl(url)).toBe(true);
    }
    for (const url of [" ", "/docs", "//example.com", "ftp://example.com"]) {
      expect(isValidOptionalSettingsHttpUrl(url)).toBe(false);
    }
  });
});
