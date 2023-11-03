package http

import (
	"bytes"
	"fmt"
	"net/http"
	"text/template"

	"github.com/poseidon/matchbox/matchbox/server"
	pb "github.com/poseidon/matchbox/matchbox/server/serverpb"
	"github.com/sirupsen/logrus"
)

const ipxeBootstrap = `#!ipxe
chain ipxe?uuid=${uuid}&mac=${mac:hexhyp}&domain=${domain}&hostname=${hostname}&serial=${serial}&arch=${buildarch:uristring}
`

// var ipxeTemplate = template.Must(template.New("iPXE config").Parse(`#!ipxe
// kernel {{.Kernel}}{{range $arg := .Args}} {{$arg}}{{end}}
// {{- range $element := .Initrd }}
// initrd {{$element}}
// {{- end}}
// boot
// `))

var defaultIpxeTemplate = template.Must(template.New("iPXE config").Parse(`#!ipxe
set menu-timeout 60000

menu Please choose how to boot
item ipxe             iPXE boot
item local  Local boot from first HDD
item exit Exit        iPXE and continue BIOS boot
choose --default local --timeout ${menu-timeout} target && goto ${target}

:ipxe
kernel {{.Kernel}}{{range $arg := .Args}} {{$arg}}{{end}}
{{- range $element := .Initrd }}
initrd {{$element}}
{{- end}}
boot

:local
sanboot --no-describe --drive 0x80

:exit
exit 1
`))

// ipxeInspect returns a handler that responds with the iPXE script to gather
// client machine data and chainload to the ipxeHandler.
func ipxeInspect() http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprint(w, ipxeBootstrap)
	}
	return http.HandlerFunc(fn)
}

// ipxeBoot returns a handler which renders the iPXE boot script for the
// requester.
func (s *Server) ipxeHandler(core server.Server) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		profile, err := profileFromContext(ctx)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"labels": labelsFromRequest(nil, req),
			}).Infof("No matching profile")
			http.NotFound(w, req)
			return
		}

		var buf bytes.Buffer
		var tpl *template.Template
		t, err := core.IPXEGet(ctx, &pb.IPXEGetRequest{Name: profile.IpxeId})
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"labels":       labelsFromRequest(nil, req),
				"profile":      profile.Id,
				"ipxeTemplate": profile.IpxeId,
			}).Infof("No matching ipxe template, using default one")
			tpl = defaultIpxeTemplate
		} else {
			// match was successful
			s.logger.WithFields(logrus.Fields{
				"labels":       labelsFromRequest(nil, req),
				"profile":      profile.Id,
				"ipxeTemplate": profile.IpxeId,
			}).Debug("Matched an iPXE config")
			tpl = template.Must(template.New("iPXE config").Parse(t))
		}

		// err = defaultIpxeTemplate.Execute(&buf, profile.Boot)
		err = tpl.Execute(&buf, profile.Boot)
		if err != nil {
			s.logger.Errorf("error rendering template: %v", err)
			http.NotFound(w, req)
			return
		}
		if _, err := buf.WriteTo(w); err != nil {
			s.logger.Errorf("error writing to response: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	return http.HandlerFunc(fn)
}
