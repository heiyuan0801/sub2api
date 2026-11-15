# Account field configuration

The ordinary Anthropic and OpenAI API-key forms each use one provider
definition for create, edit and the supported Base URL field in bulk edit.
The registry excludes OAuth, setup tokens, Bedrock, service accounts and
other providers from this credential renderer. Bulk editing deliberately
has no key input. Provider-specific options, such as OpenAI passthrough or
endpoint capabilities, remain in the existing form controllers.

Scheduling definitions are provider-neutral. They describe the existing
field types, limits and per-operation defaults/steps. Keep provider-specific
initialization, bulk field enable switches and rate-sync controls in the
calling form. The small payload helpers preserve the original common request
fragments; existing provider branches still own the complete payload,
preflight confirmations, requests and notifications.
