package database

import (
	"database/sql"
	"fmt"
	"forum/internal/models"
	"log"
	"strings"
	"time"
)

type PostRepoImpl struct {
	db *sql.DB
}

func CreateNewPostDB(db *sql.DB) *PostRepoImpl {
	return &PostRepoImpl{db}
}

func (postObj *PostRepoImpl) CreatePostRepo(post *models.Post) (int64, error) {
	result, err := postObj.db.Exec(`
		INSERT INTO posts (user_id, title, content, created_time, likes_counter, dislikes_counter, image_path, is_approved, reports, report_category) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		post.UserID, post.Title, post.Content, post.CreatedTime, post.LikesCounter, post.DislikeCounter, post.ImagePath, post.IsApproved, post.ReportStatus, post.ReportCategories)
	if err != nil {
		return -1, err
	}
	return result.LastInsertId()
}

func (postObj *PostRepoImpl) GetAllPosts() ([]*models.Post, error) {
	posts := []*models.Post{}
	rows, err := postObj.db.Query("SELECT * FROM posts ORDER BY created_time DESC")
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var post models.Post
		err = rows.Scan(&post.PostID, &post.UserID, &post.Title, &post.Content, &post.CreatedTime, &post.LikesCounter, &post.DislikeCounter, &post.ImagePath, &post.IsApproved, &post.ReportStatus, &post.ReportCategories)
		if err != nil {
			fmt.Println("Scanning from DB")
			return nil, err
		}
		posts = append(posts, &post)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return posts, nil
}

func (postObj *PostRepoImpl) GetCategoriesByPostID(postID int) ([]string, error) {
	categories := []string{}
	rows, err := postObj.db.Query("SELECT category_name FROM post_category WHERE post_id = ?", postID)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var category string
		if err = rows.Scan(&category); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}

func (postObj *PostRepoImpl) CreatePostCategory(categories []string, postID int) (int64, error) {
	var err error
	var result sql.Result

	for _, category := range categories {
		result, err = postObj.db.Exec(`
		INSERT INTO post_category (category_name, post_id) VALUES (?, ?);`,
			category, postID)
		if err != nil {
			return -1, err
		}

	}
	return result.LastInsertId()
}

func (postObj *PostRepoImpl) UpdateLikesCounter(postID, valueToAdd int) error {
	_, err := postObj.db.Exec("UPDATE posts SET likes_counter = likes_counter + ? WHERE id = ?", valueToAdd, postID)
	if err != nil {
		return err
	}
	return nil
}

func (postObj *PostRepoImpl) UpdateDislikesCounter(postID, valueToAdd int) error {
	_, err := postObj.db.Exec("UPDATE posts SET dislikes_counter = dislikes_counter + ? WHERE id = ?", valueToAdd, postID)
	if err != nil {
		return err
	}
	return nil
}

func (postObj *PostRepoImpl) GetReaction(postID, userID int) (int, error) {
	var reaction int
	if err := postObj.db.QueryRow(
		`SELECT reaction FROM post_votes WHERE post_id = ? AND user_id = ?`,
		postID, userID).Scan(&reaction); err != nil {
		return 0, err
	}
	return reaction, nil
}

func (postObj *PostRepoImpl) AddReactionToPostVotes(postID, userID, reaction int) error {
	_, err := postObj.db.Exec(`
		INSERT INTO post_votes (post_id, user_id,reaction, created_at, is_seen) VALUES (?, ?, ?);`,
		postID, userID, reaction)
	if err != nil {
		return err
	}
	return nil
}

func (postObj *PostRepoImpl) DeleteFromPostVotes(postID, userID int) error {
	_, err := postObj.db.Exec("DELETE FROM post_votes WHERE post_id = ? AND user_id = ?", postID, userID)
	if err != nil {
		return err
	}
	return nil
}

func (postObj *PostRepoImpl) UpdateReactionInPostVotes(postID, userID, newReaction int) error {
	_, err := postObj.db.Exec("UPDATE post_votes SET reaction = ? WHERE post_id = ? AND user_id = ?", newReaction, postID, userID)
	if err != nil {
		return err
	}
	return nil
}

func (postObj *PostRepoImpl) GetPostByID(postID int) (*models.Post, error) {
	post := &models.Post{}

	if err := postObj.db.QueryRow(
		`SELECT id, user_id, title, content, created_time, likes_counter, dislikes_counter, image_path FROM posts WHERE id = ?`,
		postID).Scan(&post.PostID, &post.UserID, &post.Title, &post.Content, &post.CreatedTime, &post.LikesCounter, &post.DislikeCounter, &post.ImagePath); err != nil {
		return nil, err
	}
	fmt.Println("Retrieved post.ImagePath:", post.ImagePath)
	return post, nil
}

func (postObj *PostRepoImpl) GetPostsByCategory(category string) ([]*models.Post, error) {
	posts := []*models.Post{}

	rows, err := postObj.db.Query(`
	SELECT * FROM posts WHERE id IN (SELECT post_id FROM post_category WHERE category_name = ?) ORDER BY created_time DESC
	`, category)
	if err != nil {
		// fmt.Println("FILTER:  1 error")
		return nil, err
	}

	for rows.Next() {
		var post models.Post
		err = rows.Scan(&post.PostID, &post.UserID, &post.Title, &post.Content, &post.CreatedTime, &post.LikesCounter, &post.DislikeCounter, &post.ImagePath, &post.IsApproved, &post.ReportStatus, &post.ReportCategories)
		if err != nil {
			return nil, err
		}
		posts = append(posts, &post)
	}
	if err = rows.Err(); err != nil {
		// fmt.Println("FILTER:  2 error")
		return nil, err
	}

	return posts, nil
}

func (postObj *PostRepoImpl) GetPostsByUserId(userID int) ([]*models.Post, error) {
	posts := []*models.Post{}
	rows, err := postObj.db.Query(`
	SELECT * FROM posts WHERE user_id = ? ORDER BY created_time DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var post models.Post
		err = rows.Scan(&post.PostID, &post.UserID, &post.Title, &post.Content, &post.CreatedTime, &post.LikesCounter, &post.DislikeCounter, &post.ImagePath, &post.IsApproved, &post.ReportStatus, &post.ReportCategories)
		if err != nil {
			return nil, err
		}
		posts = append(posts, &post)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return posts, nil
}

func (postObj *PostRepoImpl) GetPostsByLikes(userID int) ([]*models.Post, error) {
	posts := []*models.Post{}
	rows, err := postObj.db.Query(`
    SELECT p.id, p.user_id, p.title, p.content, p.created_time, p.likes_counter, 
           p.dislikes_counter, p.image_path, p.is_approved, p.reports, p.report_category
    FROM posts p 
    WHERE p.id IN (SELECT post_id FROM post_votes WHERE user_id = ?) 
    ORDER BY p.created_time DESC
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var post models.Post
		err = rows.Scan(
			&post.PostID,
			&post.UserID,
			&post.Title,
			&post.Content,
			&post.CreatedTime,
			&post.LikesCounter,
			&post.DislikeCounter,
			&post.ImagePath,
			&post.IsApproved,
			&post.ReportStatus,     // maps to 'reports' column
			&post.ReportCategories, // maps to 'report_category' column
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, &post)
	}
	return posts, nil
}

func (postObj *PostRepoImpl) DeletePostByID(postID int) error {
	_, err := postObj.db.Exec("DELETE FROM posts WHERE id = ? ", postID)
	if err != nil {
		return err
	}
	return nil
}

func (postObj *PostRepoImpl) DeletePostCategoryByPostID(postID int) error {
	_, err := postObj.db.Exec("DELETE FROM post_category WHERE post_id = ? ", postID)
	if err != nil {
		return err
	}
	return nil
}

func (postObj *PostRepoImpl) DeleteAllPostVotesByPostID(postID int) error {
	_, err := postObj.db.Exec("DELETE FROM post_votes WHERE post_id = ? ", postID)
	if err != nil {
		return err
	}
	return nil
}

func (postObj *PostRepoImpl) UpdateIsApprovePostStatus(postID int) error {
	_, err := postObj.db.Exec("UPDATE posts SET is_approved = 1 WHERE id = ?", postID)
	if err != nil {
		return err
	}
	return nil
}

func (postObj *PostRepoImpl) ChangeReportStatusOfPostbyPostID(postID int, reportStatusValue int) error {
	_, err := postObj.db.Exec("UPDATE posts SET reports = ? WHERE id = ?", reportStatusValue, postID)
	if err != nil {
		return err
	}
	return nil
}

func (postObj *PostRepoImpl) AddPostReportCategory(postID int, reportCategory string) error {
	_, err := postObj.db.Exec("UPDATE posts SET report_category = ? WHERE id = ?", reportCategory, postID)
	if err != nil {
		return err
	}
	return nil
}

func (postObj *PostRepoImpl) GetAllCategories() ([]*models.Category, error) {
	categories := []*models.Category{}
	rows, err := postObj.db.Query("SELECT * FROM categories")
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var category models.Category
		err = rows.Scan(&category.CategoryID, &category.Category)
		if err != nil {
			fmt.Println("Scanning from DB")
			return nil, err
		}
		categories = append(categories, &category)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}

func (postObj *PostRepoImpl) DeletePostCategoryByCategoryID(CategoryID int) error {
	_, err := postObj.db.Exec("DELETE FROM categories WHERE id = ? ", CategoryID)
	if err != nil {
		return err
	}
	return nil
}

func (postObj *PostRepoImpl) CreateCategory(categoryName string) (int64, error) {
	result, err := postObj.db.Exec(`
	INSERT INTO categories (category_name) VALUES (?);`,
		categoryName)
	if err != nil {
		fmt.Println("REPO LEVEL")
		return -1, err
	}
	return result.LastInsertId()
}

func (postObj *PostRepoImpl) UpdatePostContentByPostID(postID int, content string) error {
	_, err := postObj.db.Exec("UPDATE posts SET content = ? WHERE id = ?", content, postID)
	if err != nil {
		return err
	}
	return nil
}

func (postObj *PostRepoImpl) GetMyReactedPosts(userID int) (map[int]int, error) {
	postToReaction := make(map[int]int)
	rows, err := postObj.db.Query(`
	SELECT post_id,reaction FROM post_votes WHERE user_id=?`, userID)
	if err != nil {
		// fmt.Println("FILTER:  1 error")
		return nil, err
	}

	for rows.Next() {
		var postId int
		var reaction int
		err = rows.Scan(&postId, &reaction)
		if err != nil {
			return nil, err
		}
		postToReaction[postId] = reaction
	}
	if err = rows.Err(); err != nil {
		// fmt.Println("FILTER:  2 error")
		return nil, err
	}

	return postToReaction, nil
}

func (postObj *PostRepoImpl) GetAllMyPostsLikedByOtherUsers(userID int) ([]*models.PostVotes, error) {
	var PostVotes []*models.PostVotes
	rows, err := postObj.db.Query(`
        SELECT pv.id,
               pv.post_id,
               pv.user_id,
               pv.reaction,
               COALESCE(pv.is_seen, 0) as is_seen,
               pv.created_at as time
        FROM post_votes pv
        JOIN posts p ON p.id = pv.post_id
        WHERE p.user_id = ?  -- You are the post owner
        ORDER BY pv.created_at DESC`, userID) // Removed the pv.user_id != ? condition
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var PostVote models.PostVotes
		var timeStr sql.NullString
		err = rows.Scan(
			&PostVote.PostVotesID,
			&PostVote.PostID,
			&PostVote.UserID,
			&PostVote.Reaction,
			&PostVote.IsSeen,
			&timeStr,
		)
		if err != nil {
			return nil, err
		}

		if timeStr.Valid {
			// Try parsing with multiple formats
			formats := []string{
				"2006-01-02 15:04:05",
				"2006-01-02 15:04:05.999999999-07:00",
				"2006-01-02 15:04:05.999999999+07:00",
			}

			var parsedTime time.Time
			var parseErr error
			for _, format := range formats {
				parsedTime, parseErr = time.Parse(format, timeStr.String)
				if parseErr == nil {
					break
				}
			}
			if parseErr != nil {
				log.Printf("Failed to parse time %s: %v", timeStr.String, parseErr)
				PostVote.Time = time.Now()
			} else {
				PostVote.Time = parsedTime
			}
		} else {
			PostVote.Time = time.Now()
		}
		PostVotes = append(PostVotes, &PostVote)
	}
	return PostVotes, nil
}

func (postObj *PostRepoImpl) GetAllMyPostsCommentedByOtherUsers(userID int) ([]*models.PostVotes, error) {
	var PostVotes []*models.PostVotes
	rows, err := postObj.db.Query(`
        SELECT c.id,
               c.post_id,
               c.user_id,
               0 as reaction,
               COALESCE(c.is_seen, 0) as is_seen,
               c.created_time as time
        FROM comments c
        JOIN posts p ON p.id = c.post_id
        WHERE p.user_id = ?  -- You are the post owner
        ORDER BY c.created_time DESC`, userID) // Removed the c.user_id != ? condition
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var PostVote models.PostVotes
		var timeStr sql.NullString
		err = rows.Scan(
			&PostVote.PostVotesID,
			&PostVote.PostID,
			&PostVote.UserID,
			&PostVote.Reaction,
			&PostVote.IsSeen,
			&timeStr,
		)
		if err != nil {
			return nil, err
		}

		if timeStr.Valid {
			// Try parsing with multiple formats
			formats := []string{
				"2006-01-02 15:04:05",
				"2006-01-02 15:04:05.999999999-07:00",
				"2006-01-02 15:04:05.999999999+07:00",
			}

			var parsedTime time.Time
			var parseErr error
			for _, format := range formats {
				parsedTime, parseErr = time.Parse(format, timeStr.String)
				if parseErr == nil {
					break
				}
			}
			if parseErr != nil {
				log.Printf("Failed to parse time %s: %v", timeStr.String, parseErr)
				PostVote.Time = time.Now()
			} else {
				PostVote.Time = parsedTime
			}
		} else {
			PostVote.Time = time.Now()
		}
		PostVotes = append(PostVotes, &PostVote)
	}
	return PostVotes, nil
}

func (postObj *PostRepoImpl) CountUnseenNotifications(userID int) (int, error) {
	var count int
	query := `
        SELECT COUNT(*) FROM (
            SELECT id FROM post_votes 
            WHERE user_id = ? AND is_seen = 0
            UNION ALL
            SELECT id FROM comments 
            WHERE user_id = ? AND is_seen = 0
        )
    `
	err := postObj.db.QueryRow(query, userID, userID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (postObj *PostRepoImpl) MarkNotificationAsSeen(notificationID int) error {
	maxRetries := 3
	var err error

	for i := 0; i < maxRetries; i++ {
		query := `
            UPDATE post_votes 
            SET is_seen = 1 
            WHERE id = ?
        `
		_, err = postObj.db.Exec(query, notificationID)
		if err == nil {
			return nil
		}

		if strings.Contains(err.Error(), "database is locked") {
			log.Printf("Database locked, attempt %d of %d, waiting before retry...", i+1, maxRetries)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		return err // If it's not a locking error, return immediately
	}

	return err
}
