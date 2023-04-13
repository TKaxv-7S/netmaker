package auth

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/gravitl/netmaker/logger"
	"github.com/gravitl/netmaker/logic"
	"github.com/gravitl/netmaker/logic/pro/netcache"
	"github.com/gravitl/netmaker/models"
	"github.com/gravitl/netmaker/servercfg"
)

// SessionHandler - called by the HTTP router when user
// is calling netclient with join/register -s parameter in order to authenticate
// via SSO mechanism by OAuth2 protocol flow.
// This triggers a session start and it is managed by the flow implemented here and callback
// When this method finishes - the auth flow has finished either OK or by timeout or any other error occured
func SessionHandler(conn *websocket.Conn) {
	defer conn.Close()

	// If reached here we have a session from user to handle...
	messageType, message, err := conn.ReadMessage()
	if err != nil {
		logger.Log(0, "Error during message reading:", err.Error())
		return
	}

	var registerMessage models.RegisterMsg
	if err = json.Unmarshal(message, &registerMessage); err != nil {
		logger.Log(0, "Failed to unmarshall data err=", err.Error())
		return
	}
	if registerMessage.RegisterHost.ID == uuid.Nil || len(registerMessage.Password) == 0 {
		logger.Log(0, "invalid host registration attempted")
		return
	}
	logger.Log(0, "user registration attempted with host:", registerMessage.RegisterHost.Name, "user:", registerMessage.User)

	req := new(netcache.CValue)
	req.Value = string(registerMessage.RegisterHost.ID.String())
	req.Network = registerMessage.Network
	req.Host = registerMessage.RegisterHost
	req.Pass = ""
	req.User = ""
	// Add any extra parameter provided in the configuration to the Authorize Endpoint request??
	stateStr := logic.RandomString(node_signin_length)
	if err := netcache.Set(stateStr, req); err != nil {
		logger.Log(0, "Failed to process sso request -", err.Error())
		return
	}
	// Wait for the user to finish his auth flow...
	timeout := make(chan bool, 1)
	answer := make(chan string, 1)
	defer close(answer)
	defer close(timeout)

	if len(registerMessage.Network) > 0 { // TODO check if network provided, if so register host to it
		logger.Log(0, "network provided on registration", registerMessage.Network)
	}

	if len(registerMessage.User) > 0 { // handle basic auth
		// verify that server supports basic auth, then authorize the request with given credentials
		// check if user is allowed to join via node sso
		// i.e. user is admin or user has network permissions
		if !servercfg.IsBasicAuthEnabled() {
			err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				logger.Log(0, "error during message writing:", err.Error())
			}
		}
		_, err := logic.VerifyAuthRequest(models.UserAuthParams{
			UserName: registerMessage.User,
			Password: registerMessage.Password,
		})
		if err != nil {
			err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				logger.Log(0, "error during message writing:", err.Error())
			}
			return
		}
		if len(registerMessage.Network) > 0 {
			_, err = isUserIsAllowed(registerMessage.User, registerMessage.Network, false)
			if err != nil {
				err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					logger.Log(0, "error during message writing:", err.Error())
				}
				return
			}
		}

		if err = netcache.Set(stateStr, req); err != nil { // give the user's host access in the DB
			logger.Log(0, "machine failed to complete join on network,", registerMessage.Network, "-", err.Error())
			return
		}
	} else { // handle SSO / OAuth
		if auth_provider == nil {
			err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				logger.Log(0, "error during message writing:", err.Error())
			}
			return
		}
		redirectUrl = fmt.Sprintf("https://%s/api/oauth/register/%s", servercfg.GetAPIConnString(), stateStr)
		err = conn.WriteMessage(messageType, []byte(redirectUrl))
		if err != nil {
			logger.Log(0, "error during message writing:", err.Error())
		}
	}

	go func() {
		for {
			cachedReq, err := netcache.Get(stateStr)
			if err != nil {
				if strings.Contains(err.Error(), "expired") {
					logger.Log(0, "timeout occurred while waiting for SSO on network", registerMessage.Network)
					timeout <- true
					break
				}
				continue
			} else if len(cachedReq.Pass) > 0 {
				logger.Log(0, "node SSO process completed for user", cachedReq.User, "on network", registerMessage.Network)
				answer <- cachedReq.Pass
				break
			}
			time.Sleep(500) // try it 2 times per second to see if auth is completed
		}
	}()

	select {
	case result := <-answer: // a read from req.answerCh has occurred
		err = conn.WriteMessage(messageType, []byte(result))
		if err != nil {
			logger.Log(0, "error during message writing:", err.Error())
		}
	case <-timeout: // the read from req.answerCh has timed out
		logger.Log(0, "authentication server time out for a node on network", registerMessage.Network)
		err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			logger.Log(0, "error during timeout message writing:", err.Error())
		}
	}
	// The entry is not needed anymore, but we will let the producer to close it to avoid panic cases
	if err = netcache.Del(stateStr); err != nil {
		logger.Log(0, "failed to remove node SSO cache entry", err.Error())
	}
	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		logger.Log(0, "write close:", err.Error())
		return
	}
}
