// CodeMirror 6 wrapper. CM6 only renders the visible viewport, so large
// payloads stay smooth. Used for request bodies and response display.
import { createEffect, onCleanup, onMount } from "solid-js";
import { EditorState } from "@codemirror/state";
import { EditorView, keymap, lineNumbers } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { json } from "@codemirror/lang-json";
import { graphql, updateSchema } from "cm6-graphql";
import type { GraphQLSchema } from "graphql";

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
    ".cm-content": { fontFamily: "var(--mono)", caretColor: "#e6e6e6" },
    ".cm-gutters": {
      backgroundColor: "transparent",
      color: "#4b5163",
      border: "none",
    },
    ".cm-activeLine": { backgroundColor: "#ffffff08" },
    ".cm-activeLineGutter": { backgroundColor: "transparent" },
    "&.cm-focused": { outline: "none" },
  },
  { dark: true }
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
      EditorState.readOnly.of(ro),
      ...(ro
        ? []
        : [
            history(),
            keymap.of([...defaultKeymap, ...historyKeymap]),
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
