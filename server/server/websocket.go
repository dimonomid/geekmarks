package server

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/juju/errors"
)

var upgrader = websocket.Upgrader{}

func (gm *GMServer) webSocketConnect(
	w http.ResponseWriter, r *http.Request, gsu getSubjUser,
) error {
	ud, err := gm.getUserAndAuthorizeByReq(r, gsu, &authzArgs{})
	if err != nil {
		return errors.Trace(err)
	}

	fmt.Println("user connected:", ud)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return errors.Trace(err)
	}

	go func() (err error) {
		defer func() {
			fmt.Printf("error: %s\n", err)
		}()
		for {
			messageType, r, err := conn.NextReader()
			if err != nil {
				return err
			}
			w, err := conn.NextWriter(messageType)
			if err != nil {
				return err
			}
			if _, err := io.Copy(w, r); err != nil {
				return err
			}
			if err := w.Close(); err != nil {
				return err
			}
		}
	}()

	return nil
}
