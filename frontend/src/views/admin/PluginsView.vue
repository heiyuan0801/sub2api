<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="space-y-3 border-b border-border/80 pb-5">
        <PageHeader
          :title="t('admin.plugins.title')"
          :description="t('admin.plugins.description')"
        >
          <template #actions>
            <input
              ref="fileInput"
              class="hidden"
              type="file"
              accept=".s2plugin,application/zip"
              @change="handleFileSelected"
            />
            <button
              type="button"
              class="btn btn-primary h-9 px-3.5 text-sm"
              :disabled="uploading"
              @click="fileInput?.click()"
            >
              <Icon name="upload" size="sm" />
              {{ uploading ? t("common.processing") : t("admin.plugins.upload") }}
            </button>
            <button
              type="button"
              class="btn btn-secondary h-9 w-9 p-0"
              :disabled="loading"
              :title="t('common.refresh')"
              @click="loadPlugins"
            >
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              <span class="sr-only">{{ t("common.refresh") }}</span>
            </button>
          </template>
        </PageHeader>
        <div class="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
          <span class="rounded-full border border-border bg-muted/50 px-2.5 py-0.5 font-medium text-foreground/80">{{
            t("admin.plugins.onlyOpenAI")
          }}</span>
          <span class="rounded-full border border-border bg-muted/50 px-2.5 py-0.5 font-medium text-foreground/80">{{
            t("admin.plugins.noAccountCoupling")
          }}</span>
        </div>
      </div>

      <p class="text-xs text-muted-foreground">
        {{ t("admin.plugins.uploadHint") }}
      </p>

      <div
        class="rounded-2xl border border-sky-500/25 bg-sky-500/10 px-4 py-3 text-sm text-sky-900 dark:text-sky-200"
      >
        <p>{{ t("admin.plugins.runtimeNotice") }}</p>
        <p class="mt-1">{{ t("admin.plugins.menuNotice") }}</p>
      </div>

      <div
        v-if="loading"
        class="grid grid-cols-1 gap-4 xl:grid-cols-2"
      >
        <div
          v-for="i in 2"
          :key="`plugin-sk-${i}`"
          class="overflow-hidden rounded-2xl border border-border/80 bg-card shadow-sm"
        >
          <div class="flex items-start justify-between gap-3 border-b border-border/70 p-5">
            <div class="space-y-2">
              <div class="flex items-center gap-2">
                <Skeleton class="h-5 w-36" />
                <Skeleton class="h-5 w-14 rounded-full" />
              </div>
              <Skeleton class="h-3.5 w-48" />
            </div>
            <Skeleton class="h-8 w-24 rounded-lg" />
          </div>
          <div class="grid grid-cols-1 gap-4 p-5 md:grid-cols-2">
            <div class="space-y-2">
              <Skeleton class="h-3.5 w-24" />
              <Skeleton class="h-5 w-28 rounded-full" />
              <Skeleton class="h-3.5 w-40" />
            </div>
            <div class="space-y-2">
              <Skeleton class="h-3.5 w-20" />
              <div class="flex gap-2">
                <Skeleton class="h-5 w-16 rounded-full" />
                <Skeleton class="h-5 w-24 rounded-full" />
              </div>
            </div>
            <Skeleton class="h-14 rounded-xl md:col-span-2" />
          </div>
          <div class="flex justify-end gap-2 border-t border-border/70 bg-muted/20 px-5 py-3.5">
            <Skeleton class="h-8 w-20 rounded-lg" />
            <Skeleton class="h-8 w-20 rounded-lg" />
          </div>
        </div>
      </div>

      <div
        v-else-if="plugins.length === 0"
        class="flex min-h-56 flex-col items-center justify-center rounded-2xl border border-dashed border-border bg-card/50 px-6 py-10 text-center"
      >
        <Icon name="cube" size="xl" class="text-muted-foreground/50" />
        <p class="mt-3 font-medium text-foreground">
          {{ t("admin.plugins.empty") }}
        </p>
        <p class="mt-1 max-w-lg text-sm text-muted-foreground">
          {{ t("admin.plugins.emptyHint") }}
        </p>
      </div>

      <div v-else class="grid grid-cols-1 gap-4 xl:grid-cols-2">
        <article
          v-for="plugin in plugins"
          :key="plugin.id"
          class="overflow-hidden rounded-2xl border border-border/80 bg-card shadow-sm transition-colors hover:border-border"
        >
          <div
            class="flex flex-wrap items-start justify-between gap-3 border-b border-border/70 p-5"
          >
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <h3
                  class="truncate text-base font-semibold tracking-tight text-foreground"
                >
                  {{ plugin.name }}
                </h3>
                <span class="font-mono text-xs tabular-nums text-muted-foreground"
                  >v{{ plugin.version }}</span
                >
                <span
                  class="rounded-full px-2.5 py-0.5 text-xs font-medium"
                  :class="stateClass(plugin.state)"
                >
                  {{ t(`admin.plugins.${plugin.state}`) }}
                </span>
              </div>
              <p class="mt-1 text-xs text-muted-foreground">
                {{ plugin.plugin_key
                }}<span v-if="plugin.author"> · {{ plugin.author }}</span>
              </p>
              <p
                v-if="plugin.description"
                class="mt-2 text-sm text-muted-foreground"
              >
                {{ plugin.description }}
              </p>
            </div>
            <button
              type="button"
              class="btn btn-secondary h-8 px-3 text-xs"
              @click="openConfiguration(plugin)"
            >
              <Icon name="cog" size="sm" />
              {{ t("admin.plugins.configure") }}
            </button>
          </div>

          <div class="grid grid-cols-1 gap-x-6 gap-y-4 p-5 md:grid-cols-2">
            <div>
              <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                {{ t("admin.plugins.compatibility") }}
              </p>
              <div class="mt-2 flex items-center gap-2">
                <span
                  class="rounded-full px-2.5 py-0.5 text-xs font-medium"
                  :class="compatibilityClass(plugin.compatibility.status)"
                >
                  {{ t(`admin.plugins.${plugin.compatibility.status}`) }}
                </span>
                <span class="text-xs text-muted-foreground">{{
                  plugin.compatibility.message
                }}</span>
              </div>
              <dl
                class="mt-3 grid grid-cols-[auto,1fr] gap-x-3 gap-y-1 text-xs"
              >
                <dt class="text-muted-foreground">
                  {{ t("admin.plugins.currentVersion") }}
                </dt>
                <dd class="font-mono tabular-nums text-foreground">
                  {{ plugin.compatibility.current_sub2api_version }}
                </dd>
                <dt class="text-muted-foreground">
                  {{ t("admin.plugins.requiredVersion") }}
                </dt>
                <dd class="font-mono tabular-nums text-foreground">
                  {{ plugin.compatibility.required_sub2api_version }}
                </dd>
                <dt class="text-muted-foreground">
                  {{ t("admin.plugins.recommendedVersion") }}
                </dt>
                <dd class="font-mono tabular-nums text-foreground">
                  {{ plugin.compatibility.recommended_sub2api_version || "-" }}
                </dd>
              </dl>
            </div>

            <div>
              <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                {{ t("admin.plugins.runtime") }}
              </p>
              <div class="mt-2 flex flex-wrap gap-2 text-xs">
                <span
                  class="rounded-full px-2.5 py-0.5 font-medium"
                  :class="
                    plugin.runtime_healthy
                      ? 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300'
                      : 'bg-muted text-muted-foreground'
                  "
                >
                  {{
                    plugin.runtime_healthy
                      ? t("admin.plugins.healthy")
                      : t("admin.plugins.unhealthy")
                  }}
                </span>
                <span
                  class="rounded-full bg-muted px-2.5 py-0.5 font-medium text-muted-foreground"
                >
                  {{ t("admin.plugins.signature") }}:
                  {{ t(`admin.plugins.${plugin.signature_status}`) }}
                </span>
              </div>
              <p
                v-if="plugin.last_error"
                class="mt-3 break-words text-xs text-destructive"
              >
                {{ plugin.last_error }}
              </p>
              <p
                v-else-if="plugin.runtime_message"
                class="mt-3 break-words text-xs text-muted-foreground"
              >
                {{ plugin.runtime_message }}
              </p>
            </div>

            <div class="rounded-xl border border-border/70 bg-muted/40 p-3.5 md:col-span-2">
              <label
                class="flex items-center justify-between gap-4 text-xs font-medium text-foreground"
              >
                <span>{{ t("admin.plugins.rollout") }}</span>
                <span class="w-11 text-right font-mono tabular-nums text-foreground"
                  >{{
                    rolloutValues[plugin.id] ?? currentRollout(plugin)
                  }}%</span
                >
              </label>
              <input
                :value="rolloutValues[plugin.id] ?? currentRollout(plugin)"
                type="range"
                min="1"
                max="100"
                step="1"
                class="mt-2 w-full accent-foreground"
                :disabled="hasEnabledBinding(plugin)"
                @input="setRollout(plugin.id, $event)"
              />
            </div>
          </div>

          <div
            class="flex flex-wrap justify-end gap-2 border-t border-border/70 bg-muted/20 px-5 py-3.5"
          >
            <button
              type="button"
              class="btn btn-secondary h-8 px-3 text-xs"
              :disabled="busyID === plugin.id"
              @click="testPlugin(plugin)"
            >
              <Icon name="beaker" size="sm" />
              {{ t("admin.plugins.test") }}
            </button>
            <button
              v-if="hasEnabledBinding(plugin)"
              type="button"
              class="btn btn-secondary h-8 px-3 text-xs"
              :disabled="busyID === plugin.id"
              @click="disablePlugin(plugin)"
            >
              <Icon name="ban" size="sm" />
              {{ t("admin.plugins.disable") }}
            </button>
            <button
              v-else
              type="button"
              class="btn btn-primary h-8 px-3 text-xs"
              :disabled="
                busyID === plugin.id ||
                plugin.state === 'starting' ||
                !plugin.compatibility.compatible
              "
              @click="enablePlugin(plugin)"
            >
              <Icon name="play" size="sm" />
              {{ t("admin.plugins.enable") }}
            </button>
            <button
              type="button"
              class="btn btn-danger h-8 px-3 text-xs"
              :disabled="busyID === plugin.id || hasEnabledBinding(plugin)"
              @click="uninstallPlugin(plugin)"
            >
              <Icon name="trash" size="sm" />
              {{ t("admin.plugins.uninstall") }}
            </button>
          </div>
        </article>
      </div>

      <BaseDialog
        :show="configPlugin !== null"
        :title="
          t('admin.plugins.configTitle', { name: configPlugin?.name || '' })
        "
        width="full"
        @close="closeConfiguration"
      >
        <div
          class="relative min-h-[520px] overflow-hidden rounded-xl border border-border/80 bg-muted/30"
          :style="{ height: `${iframeHeight}px` }"
        >
          <div
            v-if="uiLoading"
            class="absolute inset-0 z-10 flex flex-col items-center justify-center gap-3 bg-background/80 text-sm text-muted-foreground backdrop-blur-xs"
          >
            <Skeleton class="h-4 w-40" />
            <span>{{ t("admin.plugins.loadingUI") }}</span>
          </div>
          <div
            v-if="uiError"
            class="absolute inset-0 z-20 flex flex-col items-center justify-center bg-card p-8 text-center"
          >
            <Icon name="exclamationTriangle" size="xl" class="text-amber-500" />
            <p class="mt-3 font-medium text-foreground">
              {{ t("admin.plugins.uiUnavailable") }}
            </p>
            <p class="mt-1 max-w-xl text-sm text-muted-foreground">{{ uiError }}</p>
          </div>
          <iframe
            v-if="uiSession"
            ref="pluginFrame"
            :src="uiSession.url"
            sandbox="allow-scripts"
            referrerpolicy="no-referrer"
            class="h-full w-full border-0 bg-background"
            :title="
              t('admin.plugins.configTitle', { name: configPlugin?.name || '' })
            "
            @load="handlePluginFrameLoad"
          />
        </div>
      </BaseDialog>

      <TotpStepUpDialog :controller="pluginStepUp" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  adminAPI,
  type PluginInstallation,
  type PluginUISession,
} from "@/api/admin";
import { useAppStore } from "@/stores";
import AppLayout from "@/components/layout/AppLayout.vue";
import BaseDialog from "@/components/common/BaseDialog.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import Icon from "@/components/icons/Icon.vue";
import { Skeleton } from "@/components/ui/skeleton";
import TotpStepUpDialog from "@/components/auth/TotpStepUpDialog.vue";
import {
  isStepUpBlocked,
  isStepUpCancelled,
  stepUpBlockReason,
  useStepUp,
} from "@/composables/useStepUp";

interface PluginBridgeMessage {
  source?: string;
  bridge_token?: string;
  type?: string;
  request_id?: string;
  config?: unknown;
  height?: unknown;
  level?: unknown;
  message?: unknown;
}

const { t } = useI18n();
const appStore = useAppStore();
const pluginStepUp = useStepUp();
const plugins = ref<PluginInstallation[]>([]);
const loading = ref(false);
const uploading = ref(false);
const busyID = ref<number | null>(null);
const fileInput = ref<HTMLInputElement | null>(null);
const rolloutValues = ref<Record<number, number>>({});
const configPlugin = ref<PluginInstallation | null>(null);
const uiSession = ref<PluginUISession | null>(null);
const pluginFrame = ref<HTMLIFrameElement | null>(null);
const uiLoading = ref(false);
const uiError = ref("");
const iframeHeight = ref(640);
const pluginFrameLoaded = ref(false);
const pendingBridgeRequests = new Map<string, number>();
let uiSessionVersion = 0;

function errorMessage(error: unknown): string {
  if (typeof error === "object" && error !== null && "message" in error) {
    return String(
      (error as { message?: unknown }).message || t("common.unknownError"),
    );
  }
  return t("common.unknownError");
}

function reportSensitiveActionError(error: unknown): void {
  if (isStepUpCancelled(error)) return;
  if (isStepUpBlocked(error)) {
    appStore.showError(
      stepUpBlockReason(error) === "STEP_UP_ADMIN_API_KEY_FORBIDDEN"
        ? t("stepUp.adminApiKeyForbidden")
        : t("stepUp.notEnabled"),
    );
    return;
  }
  appStore.showError(errorMessage(error));
}

async function loadPlugins(): Promise<void> {
  loading.value = true;
  try {
    plugins.value = await adminAPI.plugins.list();
    for (const plugin of plugins.value) {
      rolloutValues.value[plugin.id] = currentRollout(plugin);
    }
  } catch (error: unknown) {
    appStore.showError(errorMessage(error));
  } finally {
    loading.value = false;
  }
}

async function handleFileSelected(event: Event): Promise<void> {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  target.value = "";
  if (!file || !file.name.toLowerCase().endsWith(".s2plugin")) {
    appStore.showError(t("admin.plugins.fileRequired"));
    return;
  }
  uploading.value = true;
  try {
    await pluginStepUp.run(() => adminAPI.plugins.upload(file));
    appStore.showSuccess(t("admin.plugins.uploadSuccess"));
    await loadPlugins();
  } catch (error: unknown) {
    reportSensitiveActionError(error);
  } finally {
    uploading.value = false;
  }
}

function currentRollout(plugin: PluginInstallation): number {
  return (
    plugin.bindings.find(
      (binding) => binding.capability === "openai.oauth.outbound_transport.v1",
    )?.rollout_percent || 100
  );
}

function hasEnabledBinding(plugin: PluginInstallation): boolean {
  return plugin.bindings.some((binding) => binding.enabled);
}

function setRollout(id: number, event: Event): void {
  const value = Number((event.target as HTMLInputElement).value);
  rolloutValues.value[id] = Math.min(100, Math.max(1, value));
}

async function enablePlugin(plugin: PluginInstallation): Promise<void> {
  let acceptUntested = false;
  if (!plugin.compatibility.tested) {
    acceptUntested = window.confirm(t("admin.plugins.confirmUntested"));
    if (!acceptUntested) return;
  }
  busyID.value = plugin.id;
  try {
    await pluginStepUp.run(() =>
      adminAPI.plugins.enable(
        plugin.id,
        rolloutValues.value[plugin.id] || 100,
        acceptUntested,
      ),
    );
    appStore.showSuccess(t("admin.plugins.enableSuccess"));
    await loadPlugins();
  } catch (error: unknown) {
    reportSensitiveActionError(error);
  } finally {
    busyID.value = null;
  }
}

async function disablePlugin(plugin: PluginInstallation): Promise<void> {
  if (!window.confirm(t("admin.plugins.confirmDisable"))) return;
  busyID.value = plugin.id;
  try {
    await pluginStepUp.run(() => adminAPI.plugins.disable(plugin.id));
    appStore.showSuccess(t("admin.plugins.disableSuccess"));
    await loadPlugins();
  } catch (error: unknown) {
    reportSensitiveActionError(error);
  } finally {
    busyID.value = null;
  }
}

async function uninstallPlugin(plugin: PluginInstallation): Promise<void> {
  if (!window.confirm(t("admin.plugins.confirmUninstall"))) return;
  busyID.value = plugin.id;
  try {
    await pluginStepUp.run(() => adminAPI.plugins.remove(plugin.id));
    appStore.showSuccess(t("admin.plugins.uninstallSuccess"));
    await loadPlugins();
  } catch (error: unknown) {
    reportSensitiveActionError(error);
  } finally {
    busyID.value = null;
  }
}

async function testPlugin(plugin: PluginInstallation): Promise<void> {
  busyID.value = plugin.id;
  try {
    const result = await pluginStepUp.run(() =>
      adminAPI.plugins.test(plugin.id),
    );
    if (result.success)
      appStore.showSuccess(result.message || t("admin.plugins.testSuccess"));
    else appStore.showError(result.message || t("common.error"));
  } catch (error: unknown) {
    reportSensitiveActionError(error);
  } finally {
    busyID.value = null;
  }
}

async function openConfiguration(plugin: PluginInstallation): Promise<void> {
  const version = ++uiSessionVersion;
  configPlugin.value = plugin;
  uiSession.value = null;
  pluginFrameLoaded.value = false;
  clearPendingBridgeRequests();
  uiLoading.value = true;
  uiError.value = "";
  iframeHeight.value = 640;
  try {
    const session = await adminAPI.plugins.createUISession(plugin.id);
    if (version !== uiSessionVersion) return;
    uiSession.value = session;
  } catch (error: unknown) {
    if (version !== uiSessionVersion) return;
    uiLoading.value = false;
    uiError.value = errorMessage(error);
  }
}

function closeConfiguration(): void {
  uiSessionVersion++;
  clearPendingBridgeRequests();
  pluginFrameLoaded.value = false;
  configPlugin.value = null;
  uiSession.value = null;
  uiLoading.value = false;
  uiError.value = "";
}

function clearPendingBridgeRequests(): void {
  for (const timeout of pendingBridgeRequests.values()) window.clearTimeout(timeout);
  pendingBridgeRequests.clear();
}

function handlePluginFrameLoad(): void {
  // A load can also be caused by a plugin navigating its iframe. Drop all
  // outstanding responses so a late config response is never sent to the new document.
  if (pluginFrameLoaded.value) clearPendingBridgeRequests();
  pluginFrameLoaded.value = true;
  uiLoading.value = false;
}

function registerBridgeRequest(requestID: string): void {
  const timeout = window.setTimeout(() => {
    pendingBridgeRequests.delete(requestID);
  }, 30_000);
  pendingBridgeRequests.set(requestID, timeout);
}

function postBridgeResult(
  request: PluginBridgeMessage,
  payload: Record<string, unknown>,
): void {
  if (!pluginFrame.value?.contentWindow || !uiSession.value) return;
  const requestID = typeof request.request_id === "string" ? request.request_id.trim() : "";
  const timeout = pendingBridgeRequests.get(requestID);
  if (!requestID || timeout === undefined) return;
  window.clearTimeout(timeout);
  pendingBridgeRequests.delete(requestID);
  pluginFrame.value.contentWindow.postMessage(
    {
      source: "sub2api-plugin-host",
      bridge_token: uiSession.value.bridge_token,
      type: `${request.type}.result`,
      request_id: requestID,
      ...payload,
    },
    // The sandboxed iframe has an opaque origin, so no fixed target origin exists.
    // Pending request tracking plus load invalidation prevents cross-navigation leaks.
    "*",
  );
}

async function handleBridgeMessage(event: MessageEvent): Promise<void> {
  if (
    !uiSession.value ||
    !configPlugin.value ||
    event.source !== pluginFrame.value?.contentWindow ||
    event.origin !== "null"
  )
    return;
  const message = event.data as PluginBridgeMessage;
  if (
    !message ||
    message.source !== "sub2api-plugin-ui" ||
    message.bridge_token !== uiSession.value.bridge_token
  )
    return;

  const requestID = typeof message.request_id === "string" ? message.request_id.trim() : "";
  const expectsResponse =
    message.type === "config.load" ||
    message.type === "config.save" ||
    message.type === "config.test" ||
    message.type === "plugin.status";
  if (expectsResponse) {
    if (!requestID || pendingBridgeRequests.has(requestID)) return;
    registerBridgeRequest(requestID);
  }

  try {
    switch (message.type) {
      case "sub2api.plugin.ready":
        uiLoading.value = false;
        break;
      case "config.load": {
        const config = await adminAPI.plugins.getConfig(configPlugin.value.id);
        postBridgeResult(message, { ok: true, config });
        break;
      }
      case "config.save": {
        if (
          !message.config ||
          typeof message.config !== "object" ||
          Array.isArray(message.config)
        ) {
          throw new Error(t("admin.plugins.bridgeRejected"));
        }
        const config = await pluginStepUp.run(() =>
          adminAPI.plugins.saveConfig(
            configPlugin.value!.id,
            message.config as Record<string, unknown>,
          ),
        );
        postBridgeResult(message, { ok: true, config });
        appStore.showSuccess(t("common.saved"));
        break;
      }
      case "config.test": {
        const result = await pluginStepUp.run(() =>
          adminAPI.plugins.test(configPlugin.value!.id),
        );
        postBridgeResult(message, { ok: result.success, result });
        // A successful result is delivered back to the plugin UI, which owns how it
        // presents it (inline status, or an explicit ui.notify). Only force a host
        // toast on failure so genuine errors are never silently dropped — plugins
        // may call config.test for lightweight status polling, not just as an
        // explicit "test" action, and those must not spam a success toast.
        if (!result.success)
          appStore.showError(result.message || t("common.error"));
        break;
      }
      case "plugin.status": {
        // Read-only runtime status (the plugin's Health snapshot). It has no side
        // effects, so it is intentionally NOT step-up gated and never raises a host
        // toast — the plugin UI renders it however it likes. This is the generic
        // channel for any plugin to surface live state without abusing config.test.
        const result = await adminAPI.plugins.status(configPlugin.value!.id);
        postBridgeResult(message, { ok: true, result });
        break;
      }
      case "ui.resize": {
        const height = Number(message.height);
        if (Number.isFinite(height))
          iframeHeight.value = Math.min(960, Math.max(520, Math.round(height)));
        break;
      }
      case "ui.notify": {
        const text =
          typeof message.message === "string"
            ? message.message.slice(0, 500)
            : "";
        if (!text) break;
        if (message.level === "error") appStore.showError(text);
        else if (message.level === "success") appStore.showSuccess(text);
        else appStore.showInfo(text);
        break;
      }
    }
  } catch (error: unknown) {
    if (isStepUpBlocked(error)) reportSensitiveActionError(error);
    postBridgeResult(message, {
      ok: false,
      error: isStepUpCancelled(error) ? t("common.cancel") : errorMessage(error),
    });
  }
}

function stateClass(state: PluginInstallation["state"]): string {
  if (state === "enabled")
    return "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300";
  if (state === "error" || state === "incompatible")
    return "bg-destructive/15 text-destructive";
  if (state === "starting")
    return "bg-amber-500/15 text-amber-700 dark:text-amber-300";
  return "bg-muted text-muted-foreground";
}

function compatibilityClass(
  status: PluginInstallation["compatibility"]["status"],
): string {
  if (status === "compatible")
    return "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300";
  if (status === "untested")
    return "bg-amber-500/15 text-amber-700 dark:text-amber-300";
  return "bg-destructive/15 text-destructive";
}

onMounted(() => {
  window.addEventListener("message", handleBridgeMessage);
  void loadPlugins();
});

onBeforeUnmount(() => {
  window.removeEventListener("message", handleBridgeMessage);
  closeConfiguration();
});
</script>
