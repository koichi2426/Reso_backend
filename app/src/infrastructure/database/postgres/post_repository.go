package postgres

import (
	"app/src/domain/entities"
	"app/src/domain/value_objects"
	"database/sql"
)

type PostRepository struct {
	db *sql.DB
}

func NewPostRepository(db *sql.DB) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) Create(post *entities.Post) (*entities.Post, error) {
	// 修正ポイント：username カラムと $3 パラメータを追加。引数の順番も整理。
	query := `
		INSERT INTO posts (user_id, spot_id, username, image_url, caption, posted_at) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id`

	var id int
	err := r.db.QueryRow(
		query,
		post.UserID.Value(),
		post.SpotID.Value(),    // 修正：post.ID ではなく post.SpotID を渡す
		post.UserName.String(), // 修正：UserName を追加
		post.ImageURL.String(),
		post.Caption.String(),
		post.PostedAt,
	).Scan(&id)

	if err != nil {
		return nil, err
	}
	post.ID, _ = value_objects.NewID(id)
	return post, nil
}

func (r *PostRepository) FindByID(id value_objects.ID) (*entities.Post, error) {
	// SELECT に username と spot_id を追加して、entities.Post の構造に合わせる
	query := `SELECT id, user_id, spot_id, username, image_url, caption, posted_at FROM posts WHERE id = $1`
	row := r.db.QueryRow(query, id.Value())

	var pid, userID, spotID int
	var userName, imageURL, caption string
	var postedAt sql.NullTime

	if err := row.Scan(&pid, &userID, &spotID, &userName, &imageURL, &caption, &postedAt); err != nil {
		return nil, err
	}

	postID, _ := value_objects.NewID(pid)
	uID, _ := value_objects.NewID(userID)
	sID, _ := value_objects.NewID(spotID)
	uname, _ := value_objects.NewUsername(userName)
	imgURL, _ := value_objects.NewImageURL(imageURL)
	capVO, _ := value_objects.NewCaption(caption)

	return &entities.Post{
		ID:       postID,
		UserID:   uID,
		SpotID:   sID,
		UserName: uname,
		ImageURL: imgURL,
		Caption:  capVO,
		PostedAt: postedAt.Time,
	}, nil
}

func (r *PostRepository) FindBySpotID(spotID value_objects.ID) ([]*entities.Post, error) {
	query := `SELECT id, user_id, spot_id, username, image_url, caption, posted_at FROM posts WHERE spot_id = $1`
	rows, err := r.db.Query(query, spotID.Value())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*entities.Post
	for rows.Next() {
		var pid, userID, sid int
		var userName, imageURL, caption string
		var postedAt sql.NullTime
		if err := rows.Scan(&pid, &userID, &sid, &userName, &imageURL, &caption, &postedAt); err != nil {
			return nil, err
		}

		pID, _ := value_objects.NewID(pid)
		uID, _ := value_objects.NewID(userID)
		sID, _ := value_objects.NewID(sid)
		uname, _ := value_objects.NewUsername(userName)
		imgURL, _ := value_objects.NewImageURL(imageURL)
		capVO, _ := value_objects.NewCaption(caption)

		posts = append(posts, &entities.Post{
			ID:       pID,
			UserID:   uID,
			SpotID:   sID,
			UserName: uname,
			ImageURL: imgURL,
			Caption:  capVO,
			PostedAt: postedAt.Time,
		})
	}
	return posts, nil
}

func (r *PostRepository) FindByUserID(userID value_objects.ID) ([]*entities.Post, error) {
	query := `SELECT id, user_id, spot_id, username, image_url, caption, posted_at FROM posts WHERE user_id = $1 ORDER BY posted_at DESC, id DESC`
	rows, err := r.db.Query(query, userID.Value())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*entities.Post
	for rows.Next() {
		var pid, uid, sid int
		var userName, imageURL, caption string
		var postedAt sql.NullTime

		if err := rows.Scan(&pid, &uid, &sid, &userName, &imageURL, &caption, &postedAt); err != nil {
			return nil, err
		}

		pID, _ := value_objects.NewID(pid)
		uID, _ := value_objects.NewID(uid)
		sID, _ := value_objects.NewID(sid)
		uname, _ := value_objects.NewUsername(userName)
		imgURL, _ := value_objects.NewImageURL(imageURL)
		capVO, _ := value_objects.NewCaption(caption)

		posts = append(posts, &entities.Post{
			ID:       pID,
			UserID:   uID,
			SpotID:   sID,
			UserName: uname,
			ImageURL: imgURL,
			Caption:  capVO,
			PostedAt: postedAt.Time,
		})
	}

	return posts, nil
}

func (r *PostRepository) Update(post *entities.Post) error {
	query := `UPDATE posts SET image_url = $1, caption = $2, posted_at = $3, username = $4 WHERE id = $5`
	_, err := r.db.Exec(query, post.ImageURL.String(), post.Caption.String(), post.PostedAt, post.UserName.String(), post.ID.Value())
	return err
}

func (r *PostRepository) Delete(id value_objects.ID) error {
	query := `DELETE FROM posts WHERE id = $1`
	_, err := r.db.Exec(query, id.Value())
	return err
}
