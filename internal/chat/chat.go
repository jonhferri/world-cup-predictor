// Package chat keeps the league chat records tidy while PocketBase collection
// rules handle membership, ownership and Global-league access control.
package chat

import (
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/leagues"
)

const (
	messagesCollection  = "league_messages"
	reactionsCollection = "league_message_reactions"
	readsCollection     = "league_chat_reads"
)

func cleanMessage(rec *core.Record) error {
	text := strings.TrimSpace(rec.GetString("text"))
	if text == "" {
		return apis.NewBadRequestError("message text is required", nil)
	}
	rec.Set("text", text)
	return nil
}

func cleanReaction(rec *core.Record) error {
	emoji := strings.TrimSpace(rec.GetString("emoji"))
	if emoji == "" {
		return apis.NewBadRequestError("emoji is required", nil)
	}
	if utf8.RuneCountInString(emoji) > 4 {
		return apis.NewBadRequestError("emoji is too long", nil)
	}
	rec.Set("emoji", emoji)
	return nil
}

type userDTO struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatarUrl"`
}

type reactionDTO struct {
	ID     string  `json:"id"`
	Emoji  string  `json:"emoji"`
	UserID string  `json:"userId"`
	User   userDTO `json:"user"`
}

type messageDTO struct {
	ID        string        `json:"id"`
	LeagueID  string        `json:"leagueId"`
	UserID    string        `json:"userId"`
	User      userDTO       `json:"user"`
	Text      string        `json:"text"`
	Created   string        `json:"created"`
	Updated   string        `json:"updated"`
	EditedAt  string        `json:"editedAt,omitempty"`
	Reactions []reactionDTO `json:"reactions"`
}

type overviewDTO struct {
	LeagueID   string      `json:"leagueId"`
	LeagueName string      `json:"leagueName"`
	Message    *messageDTO `json:"message"`
	Unread     int         `json:"unread"`
	LastReadAt string      `json:"lastReadAt,omitempty"`
}

func dateString(rec *core.Record, field string) string {
	dt := rec.GetDateTime(field)
	if dt.IsZero() {
		return ""
	}
	return dt.Time().Format(time.RFC3339Nano)
}

func avatarURL(user *core.Record) *string {
	file := user.GetString("avatar")
	if file == "" {
		return nil
	}
	url := "/api/files/users/" + user.Id + "/" + file
	return &url
}

func userInfo(app core.App, userID string, cache map[string]userDTO) userDTO {
	if u, ok := cache[userID]; ok {
		return u
	}
	u := userDTO{ID: userID, Name: "Ukjend spelar"}
	if rec, err := app.FindRecordById("users", userID); err == nil {
		u.Name = strings.TrimSpace(rec.GetString("name"))
		if u.Name == "" {
			u.Name = "Spelar"
		}
		u.AvatarURL = avatarURL(rec)
	}
	cache[userID] = u
	return u
}

func requireLeagueMember(app core.App, leagueID, userID string) (*core.Record, error) {
	league, err := app.FindRecordById("leagues", leagueID)
	if err != nil {
		return nil, apis.NewNotFoundError("league not found", nil)
	}
	if league.GetString("inviteCode") == leagues.GlobalInviteCode {
		return nil, apis.NewForbiddenError("global league has no chat", nil)
	}
	if _, err := app.FindFirstRecordByFilter("league_members",
		"league = {:l} && user = {:u}",
		map[string]any{"l": leagueID, "u": userID}); err != nil {
		return nil, apis.NewForbiddenError("not a member of this league", nil)
	}
	return league, nil
}

func reactionsFor(app core.App, messageID string, users map[string]userDTO) ([]reactionDTO, error) {
	recs, err := app.FindRecordsByFilter(reactionsCollection,
		"message = {:m}", "created", 0, 0, map[string]any{"m": messageID})
	if err != nil {
		return nil, err
	}
	out := make([]reactionDTO, 0, len(recs))
	for _, rec := range recs {
		uid := rec.GetString("user")
		out = append(out, reactionDTO{
			ID:     rec.Id,
			Emoji:  rec.GetString("emoji"),
			UserID: uid,
			User:   userInfo(app, uid, users),
		})
	}
	return out, nil
}

func messageInfo(app core.App, rec *core.Record, users map[string]userDTO) (messageDTO, error) {
	reactions, err := reactionsFor(app, rec.Id, users)
	if err != nil {
		return messageDTO{}, err
	}
	uid := rec.GetString("user")
	return messageDTO{
		ID:        rec.Id,
		LeagueID:  rec.GetString("league"),
		UserID:    uid,
		User:      userInfo(app, uid, users),
		Text:      rec.GetString("text"),
		Created:   dateString(rec, "created"),
		Updated:   dateString(rec, "updated"),
		EditedAt:  dateString(rec, "editedAt"),
		Reactions: reactions,
	}, nil
}

func listMessages(app core.App, leagueID string, limit int) ([]messageDTO, time.Time, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	recs, err := app.FindRecordsByFilter(messagesCollection,
		"league = {:l}", "-created", limit, 0, map[string]any{"l": leagueID})
	if err != nil {
		return nil, time.Time{}, err
	}
	var latest time.Time
	if len(recs) > 0 {
		latest = recs[0].GetDateTime("created").Time()
	}
	users := map[string]userDTO{}
	out := make([]messageDTO, 0, len(recs))
	for i := len(recs) - 1; i >= 0; i-- {
		msg, err := messageInfo(app, recs[i], users)
		if err != nil {
			return nil, time.Time{}, err
		}
		out = append(out, msg)
	}
	return out, latest, nil
}

func lastReadTime(app core.App, leagueID, userID string) time.Time {
	rec, err := app.FindFirstRecordByFilter(readsCollection,
		"league = {:l} && user = {:u}",
		map[string]any{"l": leagueID, "u": userID})
	if err != nil {
		return time.Time{}
	}
	return rec.GetDateTime("lastReadAt").Time()
}

func markRead(app core.App, leagueID, userID string, at time.Time) error {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	col, err := app.FindCollectionByNameOrId(readsCollection)
	if err != nil {
		return err
	}
	rec, err := app.FindFirstRecordByFilter(readsCollection,
		"league = {:l} && user = {:u}",
		map[string]any{"l": leagueID, "u": userID})
	if err != nil {
		rec = core.NewRecord(col)
		rec.Set("league", leagueID)
		rec.Set("user", userID)
	}
	current := rec.GetDateTime("lastReadAt").Time()
	if !current.IsZero() && current.After(at) {
		return nil
	}
	rec.Set("lastReadAt", at.UTC())
	return app.Save(rec)
}

func unreadCount(app core.App, leagueID, userID string, since time.Time) (int, error) {
	params := dbx.Params{"league": leagueID, "user": userID}
	filter := "league = {:league} AND user != {:user}"
	if !since.IsZero() {
		params["since"] = since.UTC().Format(time.RFC3339Nano)
		filter += " AND created > {:since}"
	}
	count, err := app.CountRecords(messagesCollection, dbx.NewExp(filter, params))
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func overview(app core.App, userID string) ([]overviewDTO, error) {
	memberships, err := app.FindRecordsByFilter("league_members",
		"user = {:u}", "", 0, 0, map[string]any{"u": userID})
	if err != nil {
		return nil, err
	}

	users := map[string]userDTO{}
	items := make([]overviewDTO, 0, len(memberships))
	for _, membership := range memberships {
		leagueID := membership.GetString("league")
		league, err := app.FindRecordById("leagues", leagueID)
		if err != nil || league.GetString("inviteCode") == leagues.GlobalInviteCode {
			continue
		}

		var latest *messageDTO
		latestRecs, err := app.FindRecordsByFilter(messagesCollection,
			"league = {:l}", "-created", 1, 0, map[string]any{"l": leagueID})
		if err != nil {
			return nil, err
		}
		if len(latestRecs) > 0 {
			msg, err := messageInfo(app, latestRecs[0], users)
			if err != nil {
				return nil, err
			}
			latest = &msg
		}

		lastRead := lastReadTime(app, leagueID, userID)
		unread, err := unreadCount(app, leagueID, userID, lastRead)
		if err != nil {
			return nil, err
		}
		item := overviewDTO{
			LeagueID:   leagueID,
			LeagueName: league.GetString("name"),
			Message:    latest,
			Unread:     unread,
		}
		if !lastRead.IsZero() {
			item.LastReadAt = lastRead.Format(time.RFC3339Nano)
		}
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		var ti, tj time.Time
		if items[i].Message != nil {
			ti, _ = time.Parse(time.RFC3339Nano, items[i].Message.Created)
		}
		if items[j].Message != nil {
			tj, _ = time.Parse(time.RFC3339Nano, items[j].Message.Created)
		}
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		if items[i].Unread != items[j].Unread {
			return items[i].Unread > items[j].Unread
		}
		return strings.ToLower(items[i].LeagueName) < strings.ToLower(items[j].LeagueName)
	})
	return items, nil
}

// Register wires chat validation hooks. Access rules live in the migration so
// realtime subscriptions and normal collection API calls share the same gate.
func Register(app core.App, se *core.ServeEvent) {
	app.OnRecordCreate(messagesCollection).BindFunc(func(e *core.RecordEvent) error {
		if err := cleanMessage(e.Record); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordUpdate(messagesCollection).BindFunc(func(e *core.RecordEvent) error {
		if err := cleanMessage(e.Record); err != nil {
			return err
		}
		e.Record.Set("editedAt", time.Now().UTC())
		return e.Next()
	})
	app.OnRecordCreate(reactionsCollection).BindFunc(func(e *core.RecordEvent) error {
		if err := cleanReaction(e.Record); err != nil {
			return err
		}
		return e.Next()
	})

	g := se.Router.Group("/api/leagues")
	g.Bind(apis.RequireAuth())

	chatGroup := se.Router.Group("/api/chat")
	chatGroup.Bind(apis.RequireAuth())
	chatGroup.GET("/overview", func(e *core.RequestEvent) error {
		items, err := overview(app, e.Auth.Id)
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"items": items})
	})

	g.GET("/{id}/chat", func(e *core.RequestEvent) error {
		leagueID := e.Request.PathValue("id")
		if _, err := requireLeagueMember(app, leagueID, e.Auth.Id); err != nil {
			return err
		}
		messages, latest, err := listMessages(app, leagueID, 50)
		if err != nil {
			return err
		}
		if err := markRead(app, leagueID, e.Auth.Id, latest); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"messages": messages})
	})

	g.POST("/{id}/chat/messages", func(e *core.RequestEvent) error {
		leagueID := e.Request.PathValue("id")
		if _, err := requireLeagueMember(app, leagueID, e.Auth.Id); err != nil {
			return err
		}
		var body struct {
			Text string `json:"text"`
		}
		if err := e.BindBody(&body); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		col, err := app.FindCollectionByNameOrId(messagesCollection)
		if err != nil {
			return err
		}
		rec := core.NewRecord(col)
		rec.Set("league", leagueID)
		rec.Set("user", e.Auth.Id)
		rec.Set("text", body.Text)
		if err := app.Save(rec); err != nil {
			return err
		}
		msg, err := messageInfo(app, rec, map[string]userDTO{})
		if err != nil {
			return err
		}
		return e.JSON(http.StatusCreated, map[string]any{"message": msg})
	})

	g.PATCH("/{id}/chat/messages/{messageId}", func(e *core.RequestEvent) error {
		leagueID := e.Request.PathValue("id")
		if _, err := requireLeagueMember(app, leagueID, e.Auth.Id); err != nil {
			return err
		}
		rec, err := app.FindRecordById(messagesCollection, e.Request.PathValue("messageId"))
		if err != nil || rec.GetString("league") != leagueID {
			return apis.NewNotFoundError("message not found", nil)
		}
		if rec.GetString("user") != e.Auth.Id {
			return apis.NewForbiddenError("only the author can edit this message", nil)
		}
		var body struct {
			Text string `json:"text"`
		}
		if err := e.BindBody(&body); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		rec.Set("text", body.Text)
		if err := app.Save(rec); err != nil {
			return err
		}
		msg, err := messageInfo(app, rec, map[string]userDTO{})
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"message": msg})
	})

	g.DELETE("/{id}/chat/messages/{messageId}", func(e *core.RequestEvent) error {
		leagueID := e.Request.PathValue("id")
		if _, err := requireLeagueMember(app, leagueID, e.Auth.Id); err != nil {
			return err
		}
		rec, err := app.FindRecordById(messagesCollection, e.Request.PathValue("messageId"))
		if err != nil || rec.GetString("league") != leagueID {
			return apis.NewNotFoundError("message not found", nil)
		}
		if rec.GetString("user") != e.Auth.Id {
			return apis.NewForbiddenError("only the author can delete this message", nil)
		}
		if err := app.Delete(rec); err != nil {
			return err
		}
		return e.NoContent(http.StatusNoContent)
	})

	g.POST("/{id}/chat/messages/{messageId}/reactions", func(e *core.RequestEvent) error {
		leagueID := e.Request.PathValue("id")
		if _, err := requireLeagueMember(app, leagueID, e.Auth.Id); err != nil {
			return err
		}
		message, err := app.FindRecordById(messagesCollection, e.Request.PathValue("messageId"))
		if err != nil || message.GetString("league") != leagueID {
			return apis.NewNotFoundError("message not found", nil)
		}
		var body struct {
			Emoji string `json:"emoji"`
		}
		if err := e.BindBody(&body); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		emoji := strings.TrimSpace(body.Emoji)
		if emoji == "" || utf8.RuneCountInString(emoji) > 4 {
			return apis.NewBadRequestError("invalid emoji", nil)
		}
		existing, _ := app.FindFirstRecordByFilter(reactionsCollection,
			"message = {:m} && user = {:u} && emoji = {:e}",
			map[string]any{"m": message.Id, "u": e.Auth.Id, "e": emoji})
		if existing != nil {
			if err := app.Delete(existing); err != nil {
				return err
			}
			return e.JSON(http.StatusOK, map[string]any{"active": false})
		}
		col, err := app.FindCollectionByNameOrId(reactionsCollection)
		if err != nil {
			return err
		}
		reaction := core.NewRecord(col)
		reaction.Set("message", message.Id)
		reaction.Set("user", e.Auth.Id)
		reaction.Set("emoji", emoji)
		if err := app.Save(reaction); err != nil {
			return err
		}
		return e.JSON(http.StatusCreated, map[string]any{"active": true})
	})
}
