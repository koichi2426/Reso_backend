package entities

import (
	"app/src/domain/value_objects"
	"time"
)

type Post struct {
	ID       value_objects.ID
	UserID   value_objects.ID
	SpotID   value_objects.ID // 追加：どのスポットに対する投稿かを識別するために必須
	UserName value_objects.Username
	ImageURL value_objects.ImageURL
	Caption  value_objects.Caption
	PostedAt time.Time
}

// NewPost の引数に spotID (int) を追加し、内部で VO に変換します
func NewPost(id, userID, spotID int, username, imageURL, caption string, postedAt time.Time) (*Post, error) {
	pid, err := value_objects.NewID(id)
	if err != nil {
		return nil, err
	}
	uid, err := value_objects.NewID(userID)
	if err != nil {
		return nil, err
	}
	// SpotID を VO に変換
	sid, err := value_objects.NewID(spotID)
	if err != nil {
		return nil, err
	}

	uname, err := value_objects.NewUsername(username)
	if err != nil {
		return nil, err
	}
	imgURL, err := value_objects.NewImageURL(imageURL)
	if err != nil {
		return nil, err
	}
	capVO, err := value_objects.NewCaption(caption)
	if err != nil {
		return nil, err
	}

	return &Post{
		ID:       pid,
		UserID:   uid,
		SpotID:   sid, // セット
		UserName: uname,
		ImageURL: imgURL,
		Caption:  capVO,
		PostedAt: postedAt,
	}, nil
}

type PostRepository interface {
	Create(post *Post) (*Post, error)
	FindByID(id value_objects.ID) (*Post, error)
	FindBySpotID(spotID value_objects.ID) ([]*Post, error)
	FindByUserID(userID value_objects.ID) ([]*Post, error)
	FindByUserIDAndMeshID(userID value_objects.ID, meshID value_objects.MeshID) ([]*Post, error)
	Update(post *Post) error
	Delete(id value_objects.ID) error
}
