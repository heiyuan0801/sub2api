import type { InjectionKey } from 'vue'

// Legacy controls normally portal to the body. Inside a Reka modal they must
// stay in its focus and pointer boundary, outside its scrollable form body.
export const legacyPortalTargetKey: InjectionKey<string> = Symbol('legacyPortalTarget')
