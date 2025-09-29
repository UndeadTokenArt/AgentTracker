package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"agentTracker/internal/hub"
	"agentTracker/internal/store"
)

var (
	wsUpgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	st         = store.New()
	hb         = hub.New()
)

func uidFromCookie(c *gin.Context) string {
	if v, err := c.Cookie("uid"); err == nil && v != "" {
		return v
	}
	v := time.Now().Format("20060102150405.000000000")
	c.SetCookie("uid", v, 86400*365, "/", "", false, true)
	return v
}

func registerRoutes(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{})
	})

	r.POST("/join", func(c *gin.Context) {
		uid := uidFromCookie(c)
		code := strings.ToUpper(strings.TrimSpace(c.PostForm("code")))
		if code == "" {
			// create new
			g := st.CreateOrGetGroup("", uid)
			c.Redirect(http.StatusSeeOther, "/g/"+g.Code)
			return
		}
		_ = st.CreateOrGetGroup(code, uid)
		c.Redirect(http.StatusSeeOther, "/g/"+code)
	})

	r.GET("/g/:code", func(c *gin.Context) {
		uid := uidFromCookie(c)
		code := strings.ToUpper(c.Param("code"))
		g, ok := st.GetGroup(code)
		if !ok {
			c.String(http.StatusNotFound, "group not found")
			return
		}
		isDM := g.DMUID == uid
		c.HTML(http.StatusOK, "group.tmpl", gin.H{"Code": code, "IsDM": isDM})
	})

	r.GET("/ws/:code", func(c *gin.Context) {
		uid := uidFromCookie(c)
		code := strings.ToUpper(c.Param("code"))
		g, ok := st.GetGroup(code)
		if !ok {
			c.String(http.StatusNotFound, "group not found")
			return
		}
		isDM := g.DMUID == uid
		conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		client := &hub.Client{Conn: conn, UID: uid, IsDM: isDM, Group: code, SendCh: make(chan []byte, 8)}
		hb.AddClient(code, client)

		// writer
		go func() {
			for msg := range client.SendCh {
				_ = conn.WriteMessage(websocket.TextMessage, msg)
			}
		}()

		// initial state
		hb.BroadcastState(code, g)

		// reader
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				break
			}
			// Minimal message router
			type Incoming struct {
				Type string                 `json:"type"`
				Data map[string]interface{} `json:"data"`
			}
			var in Incoming
			if err := json.Unmarshal(data, &in); err != nil {
				continue
			}
			switch in.Type {
			case "addPlayer":
				name := strings.TrimSpace(getStr(in.Data, "name"))
				init := getInt(in.Data, "initiative")
				bonus := getInt(in.Data, "bonus")
				if name == "" {
					name = "Player"
				}
				st.AddPlayer(code, uid, name, init, bonus)
			case "addPlayerRoll":
				name := strings.TrimSpace(getStr(in.Data, "name"))
				bonus := getInt(in.Data, "bonus")
				if name == "" {
					name = "Player"
				}
				st.AddPlayerWithRoll(code, uid, name, bonus)
			case "addMonster":
				if !isDM {
					break
				}
				name := strings.TrimSpace(getStr(in.Data, "name"))
				hp := getInt(in.Data, "hp")
				init := getInt(in.Data, "initiative")
				bonus := getInt(in.Data, "bonus")
				if name == "" {
					name = "Monster"
				}
				st.AddMonster(code, uid, name, hp, bonus, init)
			case "damage":
				if !isDM {
					break
				}
				id := getStr(in.Data, "id")
				dmg := getInt(in.Data, "dmg")
				st.DamageMonster(code, uid, id, dmg)
			case "reorder":
				if !isDM {
					break
				}
				order := getStringSlice(in.Data, "order")
				st.Reorder(code, uid, order)
			case "next":
				st.NextTurn(code)
			}
			if g2, ok := st.GetGroup(code); ok {
				hb.BroadcastState(code, g2)
			}
		}
		hb.RemoveClient(code, client)
		_ = conn.Close()
	})
}

func getStr(m map[string]interface{}, k string) string {
	if v, ok := m[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
func getInt(m map[string]interface{}, k string) int {
	if v, ok := m[k]; ok {
		switch t := v.(type) {
		case float64:
			return int(t)
		case int:
			return t
		case string:
			// ignore parse for brevity
		}
	}
	return 0
}
func getStringSlice(m map[string]interface{}, k string) []string {
	var out []string
	if v, ok := m[k]; ok {
		if arr, ok := v.([]interface{}); ok {
			for _, x := range arr {
				if s, ok := x.(string); ok {
					out = append(out, s)
				}
			}
		}
	}
	return out
}
