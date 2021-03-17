package view

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/cli/cli/internal/ghrepo"
	"github.com/cli/cli/pkg/cmd/workflow/shared"
	"github.com/cli/cli/pkg/cmdutil"
	"github.com/cli/cli/pkg/httpmock"
	"github.com/cli/cli/pkg/iostreams"
	"github.com/cli/cli/pkg/prompt"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdView(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		tty      bool
		wants    ViewOptions
		wantsErr bool
	}{
		// TODO web flag

		{
			name: "blank tty",
			tty:  true,
			wants: ViewOptions{
				Prompt: true,
			},
		},
		{
			name:     "blank nontty",
			wantsErr: true,
		},
		{
			name: "arg tty",
			cli:  "123",
			tty:  true,
			wants: ViewOptions{
				WorkflowID: "123",
			},
		},
		{
			name: "arg nontty",
			cli:  "123",
			wants: ViewOptions{
				WorkflowID: "123",
				Raw:        true,
			},
		},
		{
			name: "web tty",
			cli:  "--web",
			tty:  true,
			wants: ViewOptions{
				Prompt: true,
				Web:    true,
			},
		},
		{
			name: "web nontty",
			cli:  "-w 123",
			wants: ViewOptions{
				Raw:        true,
				Web:        true,
				WorkflowID: "123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			io, _, _, _ := iostreams.Test()
			io.SetStdinTTY(tt.tty)
			io.SetStdoutTTY(tt.tty)

			f := &cmdutil.Factory{
				IOStreams: io,
			}

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			var gotOpts *ViewOptions
			cmd := NewCmdView(f, func(opts *ViewOptions) error {
				gotOpts = opts
				return nil
			})
			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(ioutil.Discard)
			cmd.SetErr(ioutil.Discard)

			_, err = cmd.ExecuteC()
			if tt.wantsErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			assert.Equal(t, tt.wants.WorkflowID, gotOpts.WorkflowID)
			assert.Equal(t, tt.wants.Prompt, gotOpts.Prompt)
			assert.Equal(t, tt.wants.Raw, gotOpts.Raw)
			assert.Equal(t, tt.wants.Web, gotOpts.Web)
		})
	}
}

func TestViewRun(t *testing.T) {
	tests := []struct {
		name       string
		opts       *ViewOptions
		httpStubs  func(*httpmock.Registry)
		askStubs   func(*prompt.AskStubber)
		tty        bool
		wantOut    string
		wantErrOut string
		wantErr    bool
	}{
		// TODO prompt, workflows
		// TODO name, prompt, unique
		// TODO name, prompt, nonunique
		// TODO name, unique
		// TODO name, nonunique
		// TODO ID, raw
		// TODO ID
		// TODO web
		{
			name: "prompt, no workflows",
			tty:  true,
			opts: &ViewOptions{
				Prompt: true,
			},
			httpStubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/actions/workflows"),
					httpmock.JSONResponse(shared.WorkflowsPayload{}))
			},
			wantErrOut: "could not fetch workflows for OWNER/REPO: no workflows are enabled",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		reg := &httpmock.Registry{}
		tt.httpStubs(reg)
		tt.opts.HttpClient = func() (*http.Client, error) {
			return &http.Client{Transport: reg}, nil
		}

		io, _, stdout, _ := iostreams.Test()
		io.SetStdoutTTY(tt.tty)
		tt.opts.IO = io
		tt.opts.BaseRepo = func() (ghrepo.Interface, error) {
			return ghrepo.FromFullName("OWNER/REPO")
		}

		as, teardown := prompt.InitAskStubber()
		defer teardown()
		if tt.askStubs != nil {
			tt.askStubs(as)
		}

		t.Run(tt.name, func(t *testing.T) {
			err := runView(tt.opts)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErrOut, err.Error())
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantOut, stdout.String())
			reg.Verify(t)
		})
	}
}
