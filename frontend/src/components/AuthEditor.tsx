// Authentication editor. Prop-driven so it serves both per-request auth and
// (later) collection-level auth. Emits a whole replaced Auth object on change.
import { Match, Show, Switch } from "solid-js";
import type { Auth } from "../lib/api";

type AuthKind =
  | "inherit"
  | "none"
  | "bearer"
  | "basic"
  | "apikey"
  | "oauth2";

const KINDS: { value: AuthKind; label: string }[] = [
  { value: "inherit", label: "Inherit" },
  { value: "none", label: "No Auth" },
  { value: "bearer", label: "Bearer" },
  { value: "basic", label: "Basic" },
  { value: "apikey", label: "API Key" },
  { value: "oauth2", label: "OAuth 2.0" },
];

export default function AuthEditor(props: {
  auth: Auth;
  onChange: (next: Auth) => void;
  // when false the "Inherit" option is hidden (e.g. collection-level auth)
  allowInherit?: boolean;
}) {
  const kind = (): AuthKind => (props.auth.type || "inherit") as AuthKind;

  const patch = (fields: Partial<Auth>) =>
    props.onChange({ ...props.auth, ...fields } as Auth);

  const field = (
    label: string,
    key: keyof Auth,
    opts: { password?: boolean; placeholder?: string } = {}
  ) => (
    <label class="auth-field">
      <span>{label}</span>
      <input
        type={opts.password ? "password" : "text"}
        placeholder={opts.placeholder ?? "{{var}} supported"}
        value={(props.auth[key] as string) ?? ""}
        onInput={(e) => patch({ [key]: e.currentTarget.value } as Partial<Auth>)}
      />
    </label>
  );

  return (
    <div class="auth-editor">
      <label class="auth-field">
        <span>Type</span>
        <select
          value={kind()}
          onChange={(e) => patch({ type: e.currentTarget.value as Auth["type"] })}
        >
          {KINDS.filter(
            (k) => props.allowInherit !== false || k.value !== "inherit"
          ).map((k) => (
            <option value={k.value}>{k.label}</option>
          ))}
        </select>
      </label>

      <Switch>
        <Match when={kind() === "inherit"}>
          <div class="empty-hint">Uses the collection's authentication.</div>
        </Match>
        <Match when={kind() === "none"}>
          <div class="empty-hint">No authentication is sent.</div>
        </Match>
        <Match when={kind() === "bearer"}>
          {field("Token", "token")}
        </Match>
        <Match when={kind() === "basic"}>
          {field("Username", "username")}
          {field("Password", "password", { password: true })}
        </Match>
        <Match when={kind() === "apikey"}>
          {field("Key", "key", { placeholder: "X-API-Key" })}
          {field("Value", "keyValue")}
          <label class="auth-field">
            <span>Add to</span>
            <select
              value={props.auth.placement || "header"}
              onChange={(e) =>
                patch({ placement: e.currentTarget.value as any })
              }
            >
              <option value="header">Header</option>
              <option value="query">Query param</option>
            </select>
          </label>
        </Match>
        <Match when={kind() === "oauth2"}>
          <label class="auth-field">
            <span>Grant</span>
            <select
              value={props.auth.grant || "client_credentials"}
              onChange={(e) => patch({ grant: e.currentTarget.value as any })}
            >
              <option value="client_credentials">Client Credentials</option>
              <option value="password">Password</option>
            </select>
          </label>
          {field("Token URL", "tokenUrl", {
            placeholder: "https://auth.example.com/oauth/token",
          })}
          {field("Client ID", "clientId")}
          {field("Client Secret", "clientSecret", { password: true })}
          {field("Scope", "scope", { placeholder: "read write (optional)" })}
          <Show when={(props.auth.grant || "") === "password"}>
            {field("Username", "oauthUsername")}
            {field("Password", "oauthPassword", { password: true })}
          </Show>
        </Match>
      </Switch>
    </div>
  );
}
