package clashapi

import (
	"net/http"

	commonJSON "github.com/sagernet/sing/common/json"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	F "github.com/sagernet/sing/common/format"

	"github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
)

type Outbound struct {
	Tag     string `json:"tag"`
	Type    string `json:"type"`
	Options string `json:"options"`
}

func outbound(server *Server, logFactory log.ObservableFactory) http.Handler {
	r := chi.NewRouter()
	r.Put("/", replaceOutbound(server, logFactory))

	return r
}

func replaceOutbound(server *Server, logFactory log.ObservableFactory) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var newOutbound Outbound
		err := render.DecodeJSON(r.Body, &newOutbound)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, ErrBadRequest)
			return
		}

		outManager := server.outbound
		_, ok := outManager.Outbound(newOutbound.Tag)
		if !ok {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, "not found tag "+newOutbound.Tag)
			return
		}

		var options any

		switch newOutbound.Type {
		case constant.TypeAnyTLS:
			anyTlsOpt := new(option.AnyTLSOutboundOptions)
			if err := commonJSON.Unmarshal([]byte(newOutbound.Options), anyTlsOpt); err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, ErrBadRequest)
				return
			}
			options = anyTlsOpt
		case constant.TypeTrojan:
			trojanOpt := new(option.TrojanOutboundOptions)
			if err := commonJSON.Unmarshal([]byte(newOutbound.Options), trojanOpt); err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, ErrBadRequest)
				return
			}
			options = trojanOpt
		case constant.TypeVLESS:

		}

		logger := logFactory.NewLogger(F.ToString("outbound/", newOutbound.Type, "[", newOutbound.Tag, "]"))
		if err := outManager.Create(server.ctx, server.router, logger, newOutbound.Tag, newOutbound.Type, options); err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, err)
			return
		}

		server.logger.Info("updated outbound[", newOutbound.Tag, "] to type ", newOutbound.Type)
		render.Status(r, http.StatusOK)
		render.JSON(w, r, "ok")
		return
	}
}
