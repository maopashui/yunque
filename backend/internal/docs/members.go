package docs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gitee.com/yunque/yunque/backend/internal/auth"
)

// UserRef is a minimal user identity for member invites (resolved outside this package).
type UserRef struct {
	ID          string
	Username    string
	DisplayName string
	AvatarColor string
}

// UserLookup resolves usernames for sharing. Implemented by the auth layer.
type UserLookup interface {
	LookupByUsername(ctx context.Context, username string) (UserRef, error)
}

type Member struct {
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	AvatarColor string    `json:"avatar_color"`
	Role        Role      `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Service) ListMembers(ctx context.Context, actorID, documentID string) ([]Member, error) {
	if _, _, err := s.requireRole(ctx, documentID, actorID, Role.CanRead); err != nil {
		return nil, err
	}
	members, err := s.store.ListMembers(ctx, documentID)
	if err != nil {
		return nil, err
	}
	for i := range members {
		members[i].AvatarColor = auth.ResolveAvatarColor(members[i].UserID, members[i].AvatarColor)
	}
	return members, nil
}

type AddMemberInput struct {
	ActorID    string
	DocumentID string
	Username   string
	Role       Role
}

func (s *Service) AddMember(ctx context.Context, in AddMemberInput) (Member, error) {
	if _, _, err := s.requireRole(ctx, in.DocumentID, in.ActorID, Role.CanManage); err != nil {
		return Member{}, err
	}
	if s.users == nil {
		return Member{}, fmt.Errorf("%w: user lookup not configured", ErrInvalidInput)
	}
	role := Role(strings.TrimSpace(string(in.Role)))
	if role != RoleEditor && role != RoleViewer {
		return Member{}, fmt.Errorf("%w: role must be editor or viewer", ErrInvalidInput)
	}
	username := strings.TrimSpace(in.Username)
	if username == "" {
		return Member{}, fmt.Errorf("%w: username required", ErrInvalidInput)
	}
	ref, err := s.users.LookupByUsername(ctx, username)
	if err != nil {
		return Member{}, err
	}
	if ref.ID == in.ActorID {
		return Member{}, fmt.Errorf("%w: cannot change own membership via invite", ErrInvalidInput)
	}
	// Do not overwrite an existing owner via invite.
	existing, err := s.store.GetDocumentRole(ctx, in.DocumentID, ref.ID)
	if err == nil && existing == RoleOwner {
		return Member{}, fmt.Errorf("%w: cannot change owner role", ErrForbidden)
	}
	if err != nil && !errors.Is(err, ErrForbidden) && !errors.Is(err, ErrNotFound) {
		return Member{}, err
	}
	now := time.Now().UTC()
	if err := s.store.UpsertMember(ctx, in.DocumentID, ref.ID, role, now); err != nil {
		return Member{}, err
	}
	return Member{
		UserID:      ref.ID,
		Username:    ref.Username,
		DisplayName: ref.DisplayName,
		AvatarColor: ref.AvatarColor,
		Role:        role,
		CreatedAt:   now,
	}, nil
}

type UpdateMemberInput struct {
	ActorID    string
	DocumentID string
	UserID     string
	Role       Role
}

func (s *Service) UpdateMemberRole(ctx context.Context, in UpdateMemberInput) (Member, error) {
	if _, _, err := s.requireRole(ctx, in.DocumentID, in.ActorID, Role.CanManage); err != nil {
		return Member{}, err
	}
	role := Role(strings.TrimSpace(string(in.Role)))
	if role != RoleEditor && role != RoleViewer {
		return Member{}, fmt.Errorf("%w: role must be editor or viewer", ErrInvalidInput)
	}
	current, err := s.store.GetDocumentRole(ctx, in.DocumentID, in.UserID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			return Member{}, ErrNotFound
		}
		return Member{}, err
	}
	if current == RoleOwner {
		return Member{}, fmt.Errorf("%w: cannot change owner role", ErrForbidden)
	}
	now := time.Now().UTC()
	if err := s.store.UpsertMember(ctx, in.DocumentID, in.UserID, role, now); err != nil {
		return Member{}, err
	}
	members, err := s.store.ListMembers(ctx, in.DocumentID)
	if err != nil {
		return Member{}, err
	}
	for _, m := range members {
		if m.UserID == in.UserID {
			return m, nil
		}
	}
	return Member{UserID: in.UserID, Role: role, CreatedAt: now}, nil
}

func (s *Service) RemoveMember(ctx context.Context, actorID, documentID, userID string) error {
	if _, _, err := s.requireRole(ctx, documentID, actorID, Role.CanManage); err != nil {
		return err
	}
	current, err := s.store.GetDocumentRole(ctx, documentID, userID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			return ErrNotFound
		}
		return err
	}
	if current == RoleOwner {
		return fmt.Errorf("%w: cannot remove owner", ErrForbidden)
	}
	return s.store.DeleteMember(ctx, documentID, userID)
}
