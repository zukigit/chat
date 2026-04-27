package services_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/zukigit/chat/backend/internal/db"
	"github.com/zukigit/chat/backend/internal/lib"
	"github.com/zukigit/chat/backend/internal/services"
	pb "github.com/zukigit/chat/backend/proto/chat"
	"google.golang.org/grpc/codes"
)

func TestCreateConversation_DM(t *testing.T) {
	sqlDB := setupTestDB(t)
	chatServer := services.NewChatServer(sqlDB, nil)
	ids := createTestUsers(t, sqlDB, "alice", "bob", "carol", "dave")
	makeFriends(t, sqlDB, ids["alice"], ids["bob"])
	makeFriends(t, sqlDB, ids["alice"], ids["carol"])
	// dave is intentionally NOT friends with alice

	cases := []struct {
		name            string
		ctx             context.Context
		membersUsername []string
		wantErr         codes.Code
	}{
		{"valid", ctxWithUser("alice", ids["alice"]), []string{"bob"}, codes.OK},
		{"idempotent", ctxWithUser("alice", ids["alice"]), []string{"bob"}, codes.OK},
		{"no auth", context.Background(), []string{"bob"}, codes.Internal},
		{"empty members", ctxWithUser("alice", ids["alice"]), []string{}, codes.InvalidArgument},
		{"too many members", ctxWithUser("alice", ids["alice"]), []string{"bob", "carol"}, codes.InvalidArgument},
		{"unknown user", ctxWithUser("alice", ids["alice"]), []string{"nobody"}, codes.NotFound},
		{"self dm", ctxWithUser("alice", ids["alice"]), []string{"alice"}, codes.InvalidArgument},
		{"not friends", ctxWithUser("alice", ids["alice"]), []string{"dave"}, codes.PermissionDenied},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := chatServer.CreateConversation(tc.ctx, &pb.CreateConversationRequest{
				IsGroup:         false,
				MembersUsername: tc.membersUsername,
			})
			if got := grpcCode(err); got != tc.wantErr {
				t.Errorf("got %v, want %v (err: %v)", got, tc.wantErr, err)
			}
		})
	}
}

func TestCreateConversation_Group(t *testing.T) {
	sqlDB := setupTestDB(t)
	chatServer := services.NewChatServer(sqlDB, nil)
	ids := createTestUsers(t, sqlDB, "alice", "bob", "carol")
	makeFriends(t, sqlDB, ids["alice"], ids["bob"])
	// carol is intentionally NOT friends with alice

	cases := []struct {
		name            string
		ctx             context.Context
		groupName       string
		membersUsername []string
		wantErr         codes.Code
	}{
		{"valid", ctxWithUser("alice", ids["alice"]), "team-chat", []string{"bob"}, codes.OK},
		{"no auth", context.Background(), "team-chat", []string{"bob"}, codes.Internal},
		{"missing name", ctxWithUser("alice", ids["alice"]), "", []string{"bob"}, codes.InvalidArgument},
		{"unknown user", ctxWithUser("alice", ids["alice"]), "team-chat", []string{"nobody"}, codes.NotFound},
		{"empty members", ctxWithUser("alice", ids["alice"]), "solo", []string{}, codes.InvalidArgument},
		{"not friends", ctxWithUser("alice", ids["alice"]), "team-chat", []string{"carol"}, codes.PermissionDenied},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := chatServer.CreateConversation(tc.ctx, &pb.CreateConversationRequest{
				IsGroup:         true,
				Name:            tc.groupName,
				MembersUsername: tc.membersUsername,
			})
			if got := grpcCode(err); got != tc.wantErr {
				t.Errorf("got %v, want %v (err: %v)", got, tc.wantErr, err)
			}
		})
	}
}

func TestSendMessage(t *testing.T) {
	sqlDB := setupTestDB(t)
	chatServer := services.NewChatServer(sqlDB, nil)
	ids := createTestUsers(t, sqlDB, "alice", "bob", "carol")
	makeFriends(t, sqlDB, ids["alice"], ids["bob"])

	// alice↔bob DM
	convResp, err := chatServer.CreateConversation(
		ctxWithUser("alice", ids["alice"]),
		&pb.CreateConversationRequest{IsGroup: false, MembersUsername: []string{"bob"}},
	)
	if err != nil {
		t.Fatalf("setup CreateConversation: %v", err)
	}
	convID := convResp.ConversationId

	cases := []struct {
		name        string
		ctx         context.Context
		convID      int64
		content     string
		messageType string
		wantErr     codes.Code
	}{
		{"valid text", ctxWithUser("alice", ids["alice"]), convID, "hello", "", codes.OK},
		{"valid explicit type", ctxWithUser("bob", ids["bob"]), convID, "hi back", "text", codes.OK},
		{"invalid message type", ctxWithUser("alice", ids["alice"]), convID, "hi", "video", codes.InvalidArgument},
		{"empty content", ctxWithUser("alice", ids["alice"]), convID, "", "text", codes.InvalidArgument},
		{"zero conversation_id", ctxWithUser("alice", ids["alice"]), 0, "hello", "", codes.InvalidArgument},
		{"non-member", ctxWithUser("carol", ids["carol"]), convID, "hello", "", codes.PermissionDenied},
		{"no auth", context.Background(), convID, "hello", "", codes.Internal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := chatServer.SendMessage(tc.ctx, &pb.SendMessageRequest{
				ConversationId: tc.convID,
				Content:        tc.content,
				MessageType:    tc.messageType,
			})
			if got := grpcCode(err); got != tc.wantErr {
				t.Errorf("got %v, want %v (err: %v)", got, tc.wantErr, err)
			}
		})
	}
}

func TestGetMessages(t *testing.T) {
	sqlDB := setupTestDB(t)
	chatServer := services.NewChatServer(sqlDB, nil)
	ids := createTestUsers(t, sqlDB, "alice", "bob", "carol")
	makeFriends(t, sqlDB, ids["alice"], ids["bob"])

	convResp, err := chatServer.CreateConversation(
		ctxWithUser("alice", ids["alice"]),
		&pb.CreateConversationRequest{IsGroup: false, MembersUsername: []string{"bob"}},
	)
	if err != nil {
		t.Fatalf("setup CreateConversation: %v", err)
	}
	convID := convResp.ConversationId

	// seed 3 messages
	aliceCtx := ctxWithUser("alice", ids["alice"])
	for _, content := range []string{"msg1", "msg2", "msg3"} {
		if _, err := chatServer.SendMessage(aliceCtx, &pb.SendMessageRequest{
			ConversationId: convID,
			Content:        content,
		}); err != nil {
			t.Fatalf("seed SendMessage %q: %v", content, err)
		}
	}

	t.Run("valid first page", func(t *testing.T) {
		resp, err := chatServer.GetMessages(aliceCtx, &pb.GetMessagesRequest{
			ConversationId: convID,
		})
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if len(resp.Messages) != 3 {
			t.Errorf("want 3 messages, got %d", len(resp.Messages))
		}
	})

	t.Run("cursor pagination", func(t *testing.T) {
		// fetch first 2, then use cursor to get the rest
		resp1, err := chatServer.GetMessages(aliceCtx, &pb.GetMessagesRequest{
			ConversationId: convID,
			Limit:          2,
		})
		if err != nil {
			t.Fatalf("page1: %v", err)
		}
		if len(resp1.Messages) != 2 {
			t.Fatalf("want 2, got %d", len(resp1.Messages))
		}
		if resp1.NextCursor == 0 {
			t.Fatal("expected non-zero next_cursor after first page")
		}

		resp2, err := chatServer.GetMessages(aliceCtx, &pb.GetMessagesRequest{
			ConversationId: convID,
			Limit:          2,
			Cursor:         resp1.NextCursor,
		})
		if err != nil {
			t.Fatalf("page2: %v", err)
		}
		if len(resp2.Messages) != 1 {
			t.Errorf("want 1 remaining message, got %d", len(resp2.Messages))
		}
		if resp2.NextCursor != 0 {
			t.Errorf("want next_cursor=0 on last page, got %d", resp2.NextCursor)
		}
	})

	t.Run("non-member denied", func(t *testing.T) {
		_, err := chatServer.GetMessages(ctxWithUser("carol", ids["carol"]), &pb.GetMessagesRequest{
			ConversationId: convID,
		})
		if got := grpcCode(err); got != codes.PermissionDenied {
			t.Errorf("got %v, want PermissionDenied", got)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		_, err := chatServer.GetMessages(context.Background(), &pb.GetMessagesRequest{
			ConversationId: convID,
		})
		if got := grpcCode(err); got != codes.Internal {
			t.Errorf("got %v, want Internal", got)
		}
	})

	t.Run("zero conversation_id", func(t *testing.T) {
		_, err := chatServer.GetMessages(aliceCtx, &pb.GetMessagesRequest{
			ConversationId: 0,
		})
		if got := grpcCode(err); got != codes.InvalidArgument {
			t.Errorf("got %v, want InvalidArgument", got)
		}
	})
}

// TestSendMessage_NatsPublish verifies that SendMessage publishes
// db.SendMessageParams to each recipient's active chat session via NATS.
func TestSendMessage_NatsPublish(t *testing.T) {
	sqlDB := setupTestDB(t)
	js := setupTestNats(t)

	notifServer := services.NewNotificationServer(sqlDB, js)
	chatServer := services.NewChatServer(sqlDB, notifServer)

	ids := createTestUsers(t, sqlDB, "alice", "bob")
	makeFriends(t, sqlDB, ids["alice"], ids["bob"])

	convResp, err := chatServer.CreateConversation(
		ctxWithUser("alice", ids["alice"]),
		&pb.CreateConversationRequest{IsGroup: false, MembersUsername: []string{"bob"}},
	)
	if err != nil {
		t.Fatalf("setup CreateConversation: %v", err)
	}
	convID := convResp.ConversationId

	// Subject where publishIfOnline will deliver bob's messages.
	bobSubject := lib.ChatSubjectPrefix + ids["bob"].String()

	// Subscribe to bob's chat subject before sending.
	bobMsgs := make(chan *nats.Msg, 1)
	sub, err := js.ChanSubscribe(bobSubject, bobMsgs)
	if err != nil {
		t.Fatalf("subscribe %s: %v", bobSubject, err)
	}
	defer sub.Unsubscribe()

	if _, err := chatServer.SendMessage(
		ctxWithUser("alice", ids["alice"]),
		&pb.SendMessageRequest{ConversationId: convID, Content: "hello bob"},
	); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	t.Run("bob receives message params via NATS", func(t *testing.T) {
		select {
		case msg := <-bobMsgs:
			var envelope lib.ChatResponseEnvelope
			if err := json.Unmarshal(msg.Data, &envelope); err != nil {
				t.Fatalf("unmarshal envelope: %v", err)
			}
			if envelope.Type != lib.ChatEventMessage {
				t.Fatalf("envelope type: got %q, want %q", envelope.Type, lib.ChatEventMessage)
			}
			var message db.Message
			if err := json.Unmarshal(envelope.Data, &message); err != nil {
				t.Fatalf("unmarshal message data: %v", err)
			}
			if message.ConversationID != convID {
				t.Errorf("conversation_id: got %d, want %d", message.ConversationID, convID)
			}
			if message.SenderID != ids["alice"] {
				t.Errorf("sender_id: got %s, want alice (%s)", message.SenderID, ids["alice"])
			}
			if message.Content != "hello bob" {
				t.Errorf("content: got %q, want %q", message.Content, "hello bob")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timeout: expected NATS message on bob's chat subject")
		}
	})

	t.Run("alice (sender) receives no chat publish", func(t *testing.T) {
		// Alice has no chat session, so nothing should be published for her.
		// Verify by checking there are no pending messages on an alice subject.
		aliceSubject := lib.ChatSubjectPrefix + ids["alice"].String()
		aliceMsgs := make(chan *nats.Msg, 1)
		aliceSub, err := js.ChanSubscribe(aliceSubject, aliceMsgs)
		if err != nil {
			t.Fatalf("subscribe alice subject: %v", err)
		}
		defer aliceSub.Unsubscribe()

		select {
		case <-aliceMsgs:
			t.Error("unexpected NATS message published for alice (sender)")
		case <-time.After(300 * time.Millisecond):
			// expected: no message
		}
	})
}

// correct notifications for all conversation members except the sender.
func TestSendMessage_Notifications(t *testing.T) {
	sqlDB := setupTestDB(t)
	notifServer := services.NewNotificationServer(sqlDB, nil) // nil = no NATS push
	chatServer := services.NewChatServer(sqlDB, notifServer)
	q := db.New(sqlDB)

	ids := createTestUsers(t, sqlDB, "alice", "bob", "carol")
	makeFriends(t, sqlDB, ids["alice"], ids["bob"])
	makeFriends(t, sqlDB, ids["alice"], ids["carol"])

	t.Run("DM: only peer gets notification", func(t *testing.T) {
		convResp, err := chatServer.CreateConversation(
			ctxWithUser("alice", ids["alice"]),
			&pb.CreateConversationRequest{IsGroup: false, MembersUsername: []string{"bob"}},
		)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		convID := convResp.ConversationId

		if _, err := chatServer.SendMessage(
			ctxWithUser("alice", ids["alice"]),
			&pb.SendMessageRequest{ConversationId: convID, Content: "hey bob"},
		); err != nil {
			t.Fatalf("SendMessage: %v", err)
		}

		// Bob must receive exactly one notification.
		bobNotifs, err := q.GetNotificationsForUser(context.Background(), ids["bob"])
		if err != nil {
			t.Fatalf("GetNotificationsForUser bob: %v", err)
		}
		if len(bobNotifs) != 1 {
			t.Fatalf("bob: want 1 notification, got %d", len(bobNotifs))
		}
		n := bobNotifs[0]
		if n.Type != db.NotificationTypeMessage {
			t.Errorf("type: got %q, want %q", n.Type, db.NotificationTypeMessage)
		}
		if !n.ReferenceID.Valid || n.ReferenceID.Int64 != convID {
			t.Errorf("reference_id: got %v, want %d", n.ReferenceID, convID)
		}
		if !n.SenderID.Valid || n.SenderID.UUID != ids["alice"] {
			t.Errorf("sender_id: got %v, want alice (%s)", n.SenderID, ids["alice"])
		}

		// Alice (sender) must receive no notification.
		aliceNotifs, err := q.GetNotificationsForUser(context.Background(), ids["alice"])
		if err != nil {
			t.Fatalf("GetNotificationsForUser alice: %v", err)
		}
		if len(aliceNotifs) != 0 {
			t.Errorf("alice: want 0 notifications, got %d", len(aliceNotifs))
		}
	})

	t.Run("group: all members except sender get notification", func(t *testing.T) {
		convResp, err := chatServer.CreateConversation(
			ctxWithUser("alice", ids["alice"]),
			&pb.CreateConversationRequest{
				IsGroup:         true,
				Name:            "test-group",
				MembersUsername: []string{"bob", "carol"},
			},
		)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		convID := convResp.ConversationId

		if _, err := chatServer.SendMessage(
			ctxWithUser("alice", ids["alice"]),
			&pb.SendMessageRequest{ConversationId: convID, Content: "hello group"},
		); err != nil {
			t.Fatalf("SendMessage: %v", err)
		}

		for _, recipient := range []string{"bob", "carol"} {
			notifs, err := q.GetNotificationsForUser(context.Background(), ids[recipient])
			if err != nil {
				t.Fatalf("GetNotificationsForUser %s: %v", recipient, err)
			}
			// Filter to this conversation's notifications only (bob may have one from the DM sub-test above).
			var groupNotifs []db.Notification
			for _, n := range notifs {
				if n.ReferenceID.Valid && n.ReferenceID.Int64 == convID {
					groupNotifs = append(groupNotifs, n)
				}
			}
			if len(groupNotifs) != 1 {
				t.Errorf("%s: want 1 group notification, got %d", recipient, len(groupNotifs))
				continue
			}
			n := groupNotifs[0]
			if n.Type != db.NotificationTypeMessage {
				t.Errorf("%s type: got %q, want %q", recipient, n.Type, db.NotificationTypeMessage)
			}
			if !n.SenderID.Valid || n.SenderID.UUID != ids["alice"] {
				t.Errorf("%s sender_id: got %v, want alice", recipient, n.SenderID)
			}
		}
	})
}

func TestGetConversations(t *testing.T) {
	sqlDB := setupTestDB(t)
	chatServer := services.NewChatServer(sqlDB, nil)
	ids := createTestUsers(t, sqlDB, "alice", "bob", "carol")
	makeFriends(t, sqlDB, ids["alice"], ids["bob"])
	makeFriends(t, sqlDB, ids["alice"], ids["carol"])

	dmResp, err := chatServer.CreateConversation(
		ctxWithUser("alice", ids["alice"]),
		&pb.CreateConversationRequest{IsGroup: false, MembersUsername: []string{"bob"}},
	)
	if err != nil {
		t.Fatalf("setup DM: %v", err)
	}

	groupResp, err := chatServer.CreateConversation(
		ctxWithUser("alice", ids["alice"]),
		&pb.CreateConversationRequest{IsGroup: true, Name: "test-group", MembersUsername: []string{"bob", "carol"}},
	)
	if err != nil {
		t.Fatalf("setup group: %v", err)
	}

	t.Run("alice sees both conversations", func(t *testing.T) {
		resp, err := chatServer.GetConversations(ctxWithUser("alice", ids["alice"]), &pb.GetConversationsRequest{})
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if len(resp.Conversations) != 2 {
			t.Fatalf("want 2 conversations, got %d", len(resp.Conversations))
		}
	})

	t.Run("bob sees both conversations", func(t *testing.T) {
		resp, err := chatServer.GetConversations(ctxWithUser("bob", ids["bob"]), &pb.GetConversationsRequest{})
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if len(resp.Conversations) != 2 {
			t.Fatalf("want 2 conversations, got %d", len(resp.Conversations))
		}
	})

	t.Run("carol sees only group", func(t *testing.T) {
		resp, err := chatServer.GetConversations(ctxWithUser("carol", ids["carol"]), &pb.GetConversationsRequest{})
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if len(resp.Conversations) != 1 {
			t.Fatalf("want 1 conversation, got %d", len(resp.Conversations))
		}
		if !resp.Conversations[0].IsGroup {
			t.Error("expected group conversation")
		}
	})

	t.Run("group has correct members", func(t *testing.T) {
		resp, err := chatServer.GetConversations(ctxWithUser("alice", ids["alice"]), &pb.GetConversationsRequest{})
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		var group *pb.ConversationResult
		for _, c := range resp.Conversations {
			if c.Id == groupResp.ConversationId {
				group = c
				break
			}
		}
		if group == nil {
			t.Fatal("group conversation not found")
		}
		if len(group.Members) != 3 {
			t.Fatalf("want 3 members, got %d", len(group.Members))
		}
	})

	t.Run("dm has correct members", func(t *testing.T) {
		resp, err := chatServer.GetConversations(ctxWithUser("bob", ids["bob"]), &pb.GetConversationsRequest{})
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		var dm *pb.ConversationResult
		for _, c := range resp.Conversations {
			if c.Id == dmResp.ConversationId {
				dm = c
				break
			}
		}
		if dm == nil {
			t.Fatal("dm conversation not found")
		}
		if len(dm.Members) != 2 {
			t.Fatalf("want 2 members, got %d", len(dm.Members))
		}
	})

	t.Run("no auth", func(t *testing.T) {
		_, err := chatServer.GetConversations(context.Background(), &pb.GetConversationsRequest{})
		if got := grpcCode(err); got != codes.Internal {
			t.Errorf("got %v, want Internal", got)
		}
	})
}
