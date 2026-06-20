package tui

import (
	"context"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"

	"senda/internal/store"
)

func loadRequestCmd(path string) tea.Cmd {
	return func() tea.Msg {
		req, err := store.ReadRequest(path)
		msg := reqLoadedMsg{req: req, path: path}
		if err != nil {
			msg.err = err.Error()
		}
		return msg
	}
}

func (m tuiModel) sendCmd() tea.Cmd {
	req, coll, env, reqPath := m.cur, m.collPath, m.envName(), m.curPath
	sess := m.session
	return func() tea.Msg {
		resp, _ := sess.Send(context.Background(), req, coll, reqPath, env)
		return respMsg{resp: resp, err: resp.Error}
	}
}

func (m tuiModel) editCmd(path string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	c := exec.Command(editor, path)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editDoneMsg{path: path, err: err}
	})
}

// refreshReqView rebuilds the request viewport content for the active tab.
func (m *tuiModel) refreshReqView() {
	if !m.loaded {
		m.reqVp.SetContent(padContent("", m.reqVp.Width(), m.reqVp.Height()))
		return
	}
	m.reqVp.SetContent(padContent(m.renderReqTab(m.reqVp.Width()), m.reqVp.Width(), m.reqVp.Height()))
}

// refreshRespView rebuilds the response viewport content for the active tab.
func (m *tuiModel) refreshRespView() {
	m.vp.SetContent(padContent(m.renderRespTab(), m.vp.Width(), m.vp.Height()))
}
