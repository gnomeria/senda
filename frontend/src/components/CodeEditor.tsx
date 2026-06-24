// CodeMirror 6 wrapper. CM6 only renders the visible viewport, so large
// payloads stay smooth. Used for request bodies and response display.
import { createEffect, onCleanup, onMount } from "solid-js";
import { EditorState } from "@codemirror/state";
import { EditorView, keymap, lineNumbers } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { autocompletion, completionKeymap } from "@codemirror/autocomplete";
import { json } from "@codemirror/lang-json";
import { graphql, updateSchema } from "cm6-graphql";
import type { GraphQLSchema } from "graphql";
import { HighlightStyle, syntaxHighlighting, bracketMatching } from "@codemirror/language";
import { tags as t } from "@lezer/highlight";

type Props = {
  value: string;
  language?: "json" | "text" | "graphql";
  readOnly?: boolean;
  onChange?: (v: string) => void;
  // GraphQL only: when set, enables schema-aware validation + autocomplete.
  // Syntax linting runs even without it.
  schema?: GraphQLSchema;
};

const theme = EditorView.theme(
  {
    "&": { backgroundColor: "transparent", height: "100%", fontSize: "13px" },
    ".cm-content": { fontFamily: "var(--mono)", caretColor: "var(--text)" },
    ".cm-gutters": {
      backgroundColor: "transparent",
      color: "var(--text-faint)",
      border: "none",
    },
    ".cm-activeLine": { backgroundColor: "var(--hover)" },
    ".cm-matchingBracket": {
      backgroundColor: "var(--accent-dim)",
      outline: "1px solid var(--accent)",
      borderRadius: "2px",
    },
    ".cm-nonmatchingBracket": { color: "var(--err)" },
    ".cm-activeLineGutter": { backgroundColor: "transparent" },
    "&.cm-focused": { outline: "none" },
    ".cm-tooltip": {
      backgroundColor: "var(--bg-elev2)",
      border: "1px solid var(--border)",
      borderRadius: "4px",
      color: "var(--text)",
    },
    ".cm-tooltip-autocomplete ul li[aria-selected]": {
      backgroundColor: "var(--accent)",
      color: "var(--accent-fg)",
    },
    ".cm-completionIcon": { color: "var(--text-dim)" },
  },
  { dark: true }
);

// Token colours for JSON + GraphQL, drawn from the senda palette so the editor
// stops rendering flat monochrome. Tags double up (e.g. propertyName covers
// JSON keys and GraphQL fields).
const highlight = syntaxHighlighting(
  HighlightStyle.define([
    { tag: [t.keyword, t.operatorKeyword], color: "var(--syn-keyword)" }, // query/mutation/fragment, true/false/null
    { tag: [t.string, t.special(t.string)], color: "var(--syn-string)" },
    { tag: [t.number, t.bool, t.null], color: "var(--syn-number)" },
    { tag: [t.propertyName, t.definition(t.propertyName)], color: "var(--syn-property)" }, // JSON keys, GraphQL fields
    { tag: [t.typeName, t.className, t.namespace], color: "var(--syn-type)" },
    { tag: [t.variableName, t.atom, t.labelName], color: "var(--syn-variable)" },
    { tag: [t.comment, t.lineComment, t.blockComment], color: "var(--text-faint)", fontStyle: "italic" },
    { tag: [t.punctuation, t.brace, t.bracket, t.separator], color: "var(--text-dim)" },
  ]),
);

export default function CodeEditor(props: Props) {
  let host!: HTMLDivElement;
  let view: EditorView | undefined;

  onMount(() => {
    // Read-only (response) editors skip line-wrapping + history: uniform line
    // height lets CM6 use fast fixed-height virtualization (no per-scroll
    // re-measure), and there's nothing to undo. Editable bodies keep both.
    const ro = !!props.readOnly;
    const extensions = [
      lineNumbers(),
      theme,
      highlight,
      bracketMatching(),
      EditorState.readOnly.of(ro),
      ...(ro
        ? []
        : [
            history(),
            autocompletion(),
            keymap.of([...completionKeymap, ...defaultKeymap, ...historyKeymap]),
            EditorView.lineWrapping,
          ]),
      ...(props.language === "json" ? [json()] : []),
      ...(props.language === "graphql" ? graphql(props.schema) : []),
      EditorView.updateListener.of((u) => {
        if (u.docChanged && props.onChange) {
          props.onChange(u.state.doc.toString());
        }
      }),
    ];

    view = new EditorView({
      state: EditorState.create({ doc: props.value, extensions }),
      parent: host,
    });
  });

  // Sync external value changes (e.g. new response, request switch) into CM.
  createEffect(() => {
    const next = props.value;
    if (!view) return;
    const current = view.state.doc.toString();
    if (current !== next) {
      view.dispatch({
        changes: { from: 0, to: current.length, insert: next },
      });
    }
  });

  // Push a freshly introspected schema into the live GraphQL editor so lint +
  // autocomplete pick it up without rebuilding the view.
  createEffect(() => {
    const s = props.schema;
    if (!view || props.language !== "graphql") return;
    updateSchema(view, s);
  });

  onCleanup(() => view?.destroy());

  return <div class="code-editor" ref={host} />;
}
