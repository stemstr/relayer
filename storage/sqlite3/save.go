package sqlite3

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fiatjaf/relayer/v2/storage"
	"github.com/nbd-wtf/go-nostr"
)

func (b *SQLite3Backend) SaveEvent(ctx context.Context, evt *nostr.Event) error {
	// react to different kinds of events
	if evt.Kind == nostr.KindSetMetadata || evt.Kind == nostr.KindContactList || (10000 <= evt.Kind && evt.Kind < 20000) {
		// delete past events from this user
		b.DB.ExecContext(ctx, `DELETE FROM event WHERE pubkey = $1 AND kind = $2`, evt.PubKey, evt.Kind)
	} else if evt.Kind == nostr.KindRecommendServer {
		// delete past recommend_server events equal to this one
		b.DB.ExecContext(ctx, `DELETE FROM event WHERE pubkey = $1 AND kind = $2 AND content = $3`,
			evt.PubKey, evt.Kind, evt.Content)
	} else if evt.Kind >= 30000 && evt.Kind < 40000 {
		// NIP-33
		d := evt.Tags.GetFirst([]string{"d"})
		if d != nil {
			tagsLike := fmt.Sprintf(`%%"d","%s"%%`, d.Value())
			b.DB.ExecContext(ctx, `DELETE FROM event WHERE pubkey = $1 AND kind = $2 AND tags LIKE $3`, evt.PubKey, evt.Kind, tagsLike)
		}
	}

	// insert
	tagsj, _ := json.Marshal(evt.Tags)
	res, err := b.DB.ExecContext(ctx, `
        INSERT INTO event (id, pubkey, created_at, kind, tags, content, sig)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, evt.ID, evt.PubKey, evt.CreatedAt, evt.Kind, tagsj, evt.Content, evt.Sig)
	if err != nil {
		return err
	}

	nr, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if nr == 0 {
		return storage.ErrDupEvent
	}

	return nil
}
