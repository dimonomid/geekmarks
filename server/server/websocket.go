package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/interror"

	"github.com/gorilla/websocket"
	"github.com/juju/errors"
)

type WebSocketRequest struct {
	Method string `json:"method"`
	// Path after user address: e.g. the replica of "/api/my/tags" is "/tags".
	Path   string                 `json:"path"`
	Values map[string]interface{} `json:"values"`
	Body   interface{}            `json:"body,omitempty"`
}

func parseWebSocketRequest(reader io.Reader) (*WebSocketRequest, error) {
	var wsr *WebSocketRequest
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&wsr)
	if err != nil {
		return nil, interror.WrapInternalError(
			err,
			errors.Errorf("invalid data"),
		)
	}

	return wsr, nil
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (gm *GMServer) webSocketConnect(
	w http.ResponseWriter,
	r *http.Request,
	gsu getSubjUser,
	wsMux webSocketMux,
) error {
	subjUser, err := gm.getUserAndAuthorizeByReq(r, gsu, &authzArgs{})
	if err != nil {
		return errors.Trace(err)
	}

	caller := getAuthnUserDataByReq(r)

	fmt.Println("subj user:", subjUser)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return errors.Trace(err)
	}

	go func() (err error) {
		defer func() {
			fmt.Printf("error: %s\n", err)
		}()
		for {
			messageType, reader, err := conn.NextReader()
			if err != nil {
				return errors.Trace(err)
			}

			resp, err := wsMux(reader, caller, subjUser)
			if err != nil {
				resp = hh.GetErrorStruct(err)
			}

			w, err := conn.NextWriter(messageType)
			if err != nil {
				return errors.Trace(err)
			}

			encoder := json.NewEncoder(w)
			err = encoder.Encode(resp)
			if err != nil {
				return errors.Trace(err)
			}
			if err := w.Close(); err != nil {
				return errors.Trace(err)
			}
		}
	}()

	return nil
}
