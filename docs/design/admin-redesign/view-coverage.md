# Sub2API 页面改造清单

以 `frontend/src/views` 为根目录，逐一核对 **112 个 Vue 文件**。清单包含路由页和目录内的子组件。

- E2 已改：41 个。
- 前轮已改：63 个；另有 3 个专用控件仅做过语义颜色迁移。
- 未单独修改：5 个；原因见对应行。

本轮静态核对源码和页面范围。浏览器、构建、lint、测试留待 F 轮。

## 根目录页面（4）

| 文件 | 状态 | 说明 |
| --- | --- | --- |
| `HomeView.vue` | E2 已改 | 本轮更新展示层。 |
| `KeyUsageView.vue` | E2 已改 | 本轮更新展示层。 |
| `ModelPlazaView.vue` | E2 已改 | 本轮更新展示层。 |
| `NotFoundView.vue` | E2 已改 | 本轮更新展示层。 |

## 管理端（70）

| 文件 | 状态 | 说明 |
| --- | --- | --- |
| `admin/AccountsView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/AnnouncementsView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/AuditLogView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/BackupView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ChannelMonitorView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ChannelsView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ClusterView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/DashboardView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/DocsManagementView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/GroupsView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/PluginsView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/PromoCodesView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ProxiesView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/RedeemView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/RewardActivitiesView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/RiskControlView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/SettingsView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/SubscriptionsView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/UsageView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/UsersView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/accounts/AccountEditSummary.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/accounts/AccountFormSheet.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/accounts/AccountsDataTable.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/accounts/platform-config/AccountCredentialField.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/affiliates/AdminAffiliateInvitesView.vue` | 未单独修改 | 仅传入 invites；实际列表由已改造的 AdminAffiliateRecordsTable 渲染。 |
| `admin/affiliates/AdminAffiliateRebatesView.vue` | 未单独修改 | 仅传入 rebates；实际列表由已改造的 AdminAffiliateRecordsTable 渲染。 |
| `admin/affiliates/AdminAffiliateRecordsTable.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/affiliates/AdminAffiliateTransfersView.vue` | 未单独修改 | 仅传入 transfers；实际列表由已改造的 AdminAffiliateRecordsTable 渲染。 |
| `admin/affiliates/AffiliateOfflineWithdrawDialog.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/cluster/ClusterDashboardLog.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/cluster/ClusterDashboardNode.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/cluster/ClusterDashboardSummary.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/cluster/ClusterDashboardTopology.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/OpsDashboard.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/LogRetentionSelect.vue` | 未单独修改 | 系统日志子控件，复用现有 Select 和语义化输入框；保留预设、自定义天数和校验。 |
| `admin/ops/components/OpsAlertEventsCard.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/OpsAlertRulesCard.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/OpsConcurrencyCard.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/OpsDashboardHeader.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/OpsDashboardSkeleton.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/OpsEmailNotificationCard.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/OpsErrorDetailModal.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/OpsErrorDetailsModal.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/OpsErrorDistributionChart.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/OpsErrorLogTable.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/OpsErrorTrendChart.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/OpsLatencyChart.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/OpsOpenAITokenStatsCard.vue` | 前轮颜色迁移 | 已接入语义颜色；保留专用布局与交互。 |
| `admin/ops/components/OpsRequestDetailsModal.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/OpsRuntimeSettingsCard.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/OpsSettingsDialog.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/OpsSwitchRateTrendChart.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/OpsSystemLogTable.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/ops/components/OpsThroughputTrendChart.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/orders/AdminOrdersView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/orders/AdminPaymentDashboardView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/orders/AdminPaymentPlansView.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/orders/PlanEditDialog.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/rewards/AdminRewardClaims.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/settings/EmailTemplateEditor.vue` | 前轮颜色迁移 | 已接入语义颜色；保留专用布局与交互。 |
| `admin/settings/OpenAIFastPolicyUserSelector.vue` | 前轮颜色迁移 | 已接入语义颜色；保留专用布局与交互。 |
| `admin/settings/sections/AgreementSection.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/settings/sections/BackupSection.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/settings/sections/EmailSection.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/settings/sections/FeaturesSection.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/settings/sections/GatewaySection.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/settings/sections/GeneralSection.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/settings/sections/PaymentSection.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/settings/sections/SecuritySection.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |
| `admin/settings/sections/UsersSection.vue` | 前轮已改 | 沿用 B 至 E1 的改造。 |

## 登录、注册与认证（13）

| 文件 | 状态 | 说明 |
| --- | --- | --- |
| `auth/DingTalkCallbackView.vue` | E2 已改 | 本轮更新展示层。 |
| `auth/DingTalkEmailCompletionView.vue` | E2 已改 | 本轮更新展示层。 |
| `auth/EmailVerifyView.vue` | E2 已改 | 本轮更新展示层。 |
| `auth/ForgotPasswordView.vue` | E2 已改 | 本轮更新展示层。 |
| `auth/LinuxDoCallbackView.vue` | E2 已改 | 本轮更新展示层。 |
| `auth/LoginView.vue` | E2 已改 | 本轮更新展示层。 |
| `auth/MCPOAuthAuthorizeView.vue` | E2 已改 | 本轮更新展示层。 |
| `auth/OAuthCallbackView.vue` | E2 已改 | 本轮更新展示层。 |
| `auth/OidcCallbackView.vue` | E2 已改 | 本轮更新展示层。 |
| `auth/RegisterView.vue` | E2 已改 | 本轮更新展示层。 |
| `auth/ResetPasswordView.vue` | E2 已改 | 本轮更新展示层。 |
| `auth/WechatCallbackView.vue` | E2 已改 | 本轮更新展示层。 |
| `auth/WechatPaymentCallbackView.vue` | E2 已改 | 本轮更新展示层。 |

## 文档与法律（2）

| 文件 | 状态 | 说明 |
| --- | --- | --- |
| `public/DocsView.vue` | E2 已改 | 本轮更新展示层。 |
| `public/LegalDocumentView.vue` | E2 已改 | 本轮更新展示层。 |

## 安装（1）

| 文件 | 状态 | 说明 |
| --- | --- | --- |
| `setup/SetupWizardView.vue` | E2 已改 | 本轮更新展示层。 |

## 用户端（22）

| 文件 | 状态 | 说明 |
| --- | --- | --- |
| `user/AffiliateView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/AirwallexPaymentView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/AvailableChannelsView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/BatchImageGuideView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/ChannelStatusV1View.vue` | E2 已改 | 本轮更新展示层。 |
| `user/ChannelStatusV2View.vue` | E2 已改 | 本轮更新展示层。 |
| `user/ChannelStatusView.vue` | 未单独修改 | 仅按功能开关选择 V1 或 V2；两个实际页面均在 E2 改造。 |
| `user/CustomPageView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/DashboardView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/KeysView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/MCPView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/PaymentQRCodeView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/PaymentResultView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/PaymentView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/ProfileView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/RedeemView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/RewardActivitiesView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/StripePaymentView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/StripePopupView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/SubscriptionsView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/UsageView.vue` | E2 已改 | 本轮更新展示层。 |
| `user/UserOrdersView.vue` | E2 已改 | 本轮更新展示层。 |
