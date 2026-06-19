// Test stand-in for the generated Wails bindings module
// `bindings/senda/internal/model/models`. The real bindings are produced by
// `wails3 generate bindings` from the Go source and are gitignored, so unit
// tests (vitest) can't resolve them. vite.config.ts aliases the bindings
// import here when running under vitest.
//
// Keep the enum values in sync with internal/model/model.go — they are the
// YAML/JSON wire values, not arbitrary strings.

export const BodyType = {
  BodyNone: "none",
  BodyJSON: "json",
  BodyRaw: "raw",
  BodyForm: "form",
  BodyMultipart: "multipart",
  BodyGraphQL: "graphql",
} as const;
export type BodyType = (typeof BodyType)[keyof typeof BodyType];

export const AuthType = {
  AuthInherit: "inherit",
  AuthNone: "none",
  AuthBearer: "bearer",
  AuthBasic: "basic",
  AuthAPIKey: "apikey",
  AuthOAuth2: "oauth2",
} as const;
export type AuthType = (typeof AuthType)[keyof typeof AuthType];

export interface KV {
  key: string;
  value: string;
  enabled: boolean;
  desc?: string;
  file?: boolean;
}

export interface Body {
  type: BodyType;
  raw?: string;
  form?: KV[];
  variables?: string;
}

export interface Auth {
  type: AuthType;
  token?: string;
  username?: string;
  password?: string;
  key?: string;
  keyValue?: string;
  placement?: string;
  grant?: string;
  tokenUrl?: string;
  clientId?: string;
  clientSecret?: string;
  scope?: string;
  oauthUsername?: string;
  oauthPassword?: string;
}

export interface Assert {
  target: string;
  op: string;
  value?: string;
  enabled: boolean;
}

export interface Request {
  name: string;
  method: string;
  url: string;
  params: KV[];
  headers: KV[];
  body: Body;
  auth: Auth;
  asserts: Assert[];
  preScript: string;
  postScript: string;
  docs: string;
}
