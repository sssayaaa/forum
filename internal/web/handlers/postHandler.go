package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"forum/internal/models"
	helpers "forum/internal/web/handlers/helpers"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

func (h *Handler) CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		const MaxImageSize = 20 * 1024 * 1024

		cookie := helpers.SessionCookieGet(r)
		if cookie == nil {
			helpers.ErrorHandler(w, http.StatusUnauthorized, errors.New("couldn't get the cookie in the Post Creation Handler"))
			return
		}
		session, err := h.service.UserServiceInterface.GetSession(cookie.Value)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("Session failed in the Post Creation Handler"))
			return
		}

		postTitle := r.FormValue("posttitle")
		const maxTitleLength = 50
		if len(postTitle) > maxTitleLength {
			// postTitle = postTitle[:maxTitleLength]
			http.Error(w, "Title is too long (maximum 50 characters)", http.StatusBadRequest)
		}

		postCategory := r.Form["preference"]

		post := &models.Post{
			UserID:     session.UserID,
			Title:      postTitle,
			Content:    r.FormValue("postcontent"),
			Categories: postCategory,
		}

		//=============================================================
		//block of code responsible for the image upload
		r.Body = http.MaxBytesReader(w, r.Body, MaxImageSize)

		err = r.ParseMultipartForm(MaxImageSize)

		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("Image size over 20 Mb"))
			return
		}

		if len(r.MultipartForm.File["files"]) != 0 {
			r.ParseForm()
			if r.MultipartForm.File["files"][0].Size > int64(MaxImageSize) {
				helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("Image size over 20 Mb"))
			}
			file := r.MultipartForm.File["files"][0] // since only one image at a time
			path, err := h.service.AddImagesToPost(file)
			if err != nil {
				helpers.ErrorHandler(w, http.StatusInternalServerError, err)
				return
			}
			post.ImagePath = path
		}
		//=============================================================
		user, err := h.service.UserServiceInterface.GetUserByUserID(post.UserID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("UserService couldn't get user"))
			return
		}
		statusCode, postId, err := h.service.PostServiceInterface.CreatePost(post, user.Role)
		post.PostID = postId
		if err != nil {
			helpers.ErrorHandler(w, statusCode, err)
			return
		}

		expTime, err := h.service.UserServiceInterface.ExtendSessionTimeout(cookie.Value)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("The Time cannot be extended"))
			return
		}
		err = helpers.SessionCookieExtend(r, w, expTime)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("The Time cannot be extended"))
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	default:
		helpers.ErrorHandler(w, http.StatusMethodNotAllowed, errors.New("Error in Post Handler"))
		return
	}
}

func (h *Handler) ReactOnPostHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		postID, err := strconv.Atoi(r.FormValue("post_id"))
		if err != nil {
			helpers.ErrorHandler(w, http.StatusBadRequest, errors.New("Conversion of postID failed"))
			return
		}

		currReaction, err := strconv.Atoi(r.FormValue("type"))
		if err != nil {
			helpers.ErrorHandler(w, http.StatusBadRequest, errors.New("Conversion of reaction type failed"))
			return
		}

		cookie := helpers.SessionCookieGet(r)
		if cookie == nil {
			helpers.ErrorHandler(w, http.StatusUnauthorized, errors.New("Cookie cannot be reseived in Post Reaction Handler"))
			return
		}

		session, err := h.service.UserServiceInterface.GetSession(cookie.Value)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}
		expTime, err := h.service.UserServiceInterface.ExtendSessionTimeout(cookie.Value)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("Cookie cannot be extended"))
			return
		}
		err = helpers.SessionCookieExtend(r, w, expTime)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}

		if err := h.service.PostServiceInterface.UpdateReaction(currReaction, postID, session.UserID); err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	default:
		helpers.ErrorHandler(w, http.StatusUnauthorized, errors.New("Error in Post Reaction Handler"))
		return
	}
}

func (h *Handler) FilterHandler(w http.ResponseWriter, r *http.Request) {
	var userGlob *models.User
	type templateData struct {
		LoggedIn      bool
		AllPosts      []*models.Post
		User          *models.User
		AllCategories []string
	}

	switch r.Method {
	case "GET":
		var userID int
		cookie := helpers.SessionCookieGet(r)
		if cookie == nil {
			userID = 0
		} else {
			session, err := h.service.UserServiceInterface.GetSession(cookie.Value)
			if err != nil {
				helpers.ErrorHandler(w, http.StatusInternalServerError, err)
				return
			}
			userID = session.UserID
			// related to session an cookies updates:
			expTime, err := h.service.UserServiceInterface.ExtendSessionTimeout(cookie.Value)
			if err != nil {
				helpers.ErrorHandler(w, http.StatusMethodNotAllowed, errors.New("Cookie cannot be extended"))
				return
			}
			err = helpers.SessionCookieExtend(r, w, expTime)
			if err != nil {
				helpers.ErrorHandler(w, http.StatusInternalServerError, err)
				return
			}
			userGlob, err = h.service.UserServiceInterface.GetUserByUserID(session.UserID)
			if err != nil {
				helpers.ErrorHandler(w, http.StatusInternalServerError, err)
				return
			}

		}

		field := getFiltersFieldFromURL(r.URL.Path)
		posts, err := h.service.PostServiceInterface.Filter(field, userID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}

		for _, post := range posts {
			// getting the username for posts
			user, err := h.service.UserServiceInterface.GetUserByUserID(post.UserID)
			if err != nil {
				helpers.ErrorHandler(w, http.StatusInternalServerError, err)
				return
			}
			post.Username = user.Username

			// changing the format of the time
			post.CreatedTimeString = post.CreatedTime.Format("Jan 2, 2006 at 15:04")

			// assigning categories to each post
			temp_categories, err := h.service.PostServiceInterface.GetCategories(post.PostID)
			if err != nil {
				helpers.ErrorHandler(w, http.StatusInternalServerError, err)
				return
			}
			post.Categories = append(post.Categories, temp_categories...)
		}
		indexPath := "internal/web/templates/index.html"

		categories, err := h.service.PostServiceInterface.GetAllCategories()
		if err != nil {
			helpers.ErrorHandler(w, http.StatusBadRequest, errors.New("PENDING USERS were not found"))
		}

		var strCategories []string
		for _, category := range categories {
			strCategories = append(strCategories, category.Category)
		}

		data := templateData{
			LoggedIn:      h.service.IsUserLoggedIn(r),
			AllPosts:      posts,
			User:          userGlob,
			AllCategories: strCategories, //[]string{"Movie", "Game", "Book", "Others"}, // Initialize AllCategories with values
		}
		helpers.RenderTemplate(w, indexPath, data)
	default:
		helpers.ErrorHandler(w, http.StatusUnauthorized, errors.New("Error in Post Reaction Handler"))
		return
	}
}

func getFiltersFieldFromURL(url string) string {
	return strings.Title(strings.TrimPrefix(url, "/filter/"))
}

func (h *Handler) DeletePostHandler(w http.ResponseWriter, r *http.Request) {
	// fmt.Println("DELETETETETEE")

	switch r.Method {
	case "POST":
		// fmt.Println("INSIDE DELETE HANDLER OF POST")
		cookie := helpers.SessionCookieGet(r)
		if cookie == nil {
			helpers.ErrorHandler(w, http.StatusUnauthorized, errors.New("couldn't get the cookie in the Post Creation Handler"))
			return
		}
		expTime, err := h.service.UserServiceInterface.ExtendSessionTimeout(cookie.Value)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("The Time cannot be extended"))
			return
		}
		err = helpers.SessionCookieExtend(r, w, expTime)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("The Time cannot be extended"))
			return
		}
		postID := r.FormValue("postId")
		intPostID, err := strconv.Atoi(postID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}
		// fmt.Println("POST ID: ", postID)
		err = h.service.PostServiceInterface.DeletePost(intPostID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("failed when was deleting the post"))
			return
		}

		err = h.service.PostServiceInterface.DeletePostCategoryByPostID(intPostID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("failed when was deleting the post category"))
			return
		}

		err = h.service.PostServiceInterface.DeleteAllPostVotesByPostID(intPostID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("failed when was deleting the votes for posts"))
			return
		}
		err = h.service.CommentServiceInterface.DeleteAllCommentVotesByPostID(intPostID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("failed when was deleting the votes for posts"))
			return
		}

		err = h.service.CommentServiceInterface.DeleteAllCommentsByPostID(intPostID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("failed when was deleting the post"))
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	default:
		helpers.ErrorHandler(w, http.StatusMethodNotAllowed, errors.New("Error in Post Handler"))
		return
	}
}

func (h *Handler) ApprovePostHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		cookie := helpers.SessionCookieGet(r)
		if cookie == nil {
			helpers.ErrorHandler(w, http.StatusUnauthorized, errors.New("Cookie cannot be reseived in ApprovePostHandler"))
			return
		}

		expTime, err := h.service.UserServiceInterface.ExtendSessionTimeout(cookie.Value)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("Cookie cannot be extended in Approval of the Post"))
			return
		}

		err = helpers.SessionCookieExtend(r, w, expTime)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}

		postID := r.FormValue("postId")
		intPostID, err := strconv.Atoi(postID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}

		err = h.service.PostServiceInterface.ApprovePost(intPostID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("failed when was deleting the votes for posts"))
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	default:
		helpers.ErrorHandler(w, http.StatusMethodNotAllowed, errors.New("Error in Moderator Request Handler"))
		return

	}
}

func (h *Handler) ReportPostHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		cookie := helpers.SessionCookieGet(r)
		if cookie == nil {
			helpers.ErrorHandler(w, http.StatusUnauthorized, errors.New("Cookie failed in the Moderator Request Handler"))
			return
		}
		expTime, err := h.service.UserServiceInterface.ExtendSessionTimeout(cookie.Value)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("Cookie cannot be extended in Approval of the Post"))
			return
		}
		err = helpers.SessionCookieExtend(r, w, expTime)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}

		postID := r.FormValue("postId")
		r.ParseForm()
		reportCategory := r.Form.Get("report")
		// r.Form["preference"]
		if reportCategory == "" {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("Plese provide select the reason for the report!"))
			return
		}
		// fmt.Println("FROM FRONT: ", reportCategory)

		intPostID, err := strconv.Atoi(postID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}

		// default value is 0, then is could be changed to 1.
		err = h.service.PostServiceInterface.ChangeReportStatusOfPostbyPostID(intPostID, 1)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("failed when was deleting the votes for posts"))
			return
		}

		err = h.service.PostServiceInterface.AddPostReportCategory(intPostID, reportCategory)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	default:
		helpers.ErrorHandler(w, http.StatusMethodNotAllowed, errors.New("Error in Moderator Request Handler"))
		return

	}
}

func (h *Handler) AnswerPostReportHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		cookie := helpers.SessionCookieGet(r)
		if cookie == nil {
			helpers.ErrorHandler(w, http.StatusUnauthorized, errors.New("Cookie failed in the Moderator Request Handler"))
			return
		}
		expTime, err := h.service.UserServiceInterface.ExtendSessionTimeout(cookie.Value)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("Cookie cannot be extended in Approval of the Post"))
			return
		}
		err = helpers.SessionCookieExtend(r, w, expTime)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}

		postID := r.FormValue("postId")
		intPostID, err := strconv.Atoi(postID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}
		// fmt.Println("POST ID: ", intPostID)

		reportStatus, err := strconv.Atoi(r.FormValue("type"))
		if err != nil {
			helpers.ErrorHandler(w, http.StatusBadRequest, errors.New("Conversion of reaction type failed"))
			return
		}
		if reportStatus == 0 {
			err = h.service.PostServiceInterface.ApprovePost(intPostID)
			if err != nil {
				helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("failed when was deleting the votes for posts"))
				return
			}
			err = h.service.PostServiceInterface.ChangeReportStatusOfPostbyPostID(intPostID, reportStatus)
			if err != nil {
				helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("failed when was deleting the votes for posts"))
				return
			}
		} else {
			err = h.service.PostServiceInterface.DeletePost(intPostID)
			if err != nil {
				helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("failed when was deleting the post"))
				return
			}

			err = h.service.PostServiceInterface.DeletePostCategoryByPostID(intPostID)
			if err != nil {
				helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("failed when was deleting the post category"))
				return
			}

			err = h.service.PostServiceInterface.DeleteAllPostVotesByPostID(intPostID)
			if err != nil {
				helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("failed when was deleting the votes for posts"))
				return
			}
			err = h.service.CommentServiceInterface.DeleteAllCommentVotesByPostID(intPostID)
			if err != nil {
				helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("failed when was deleting the votes for posts"))
				return
			}

			err = h.service.CommentServiceInterface.DeleteAllCommentsByPostID(intPostID)
			if err != nil {
				helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("failed when was deleting the post"))
				return
			}
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	default:
		helpers.ErrorHandler(w, http.StatusMethodNotAllowed, errors.New("Error in Moderator Request Handler"))
		return
	}
}

func (h *Handler) EditPostHandler(w http.ResponseWriter, r *http.Request) {
	// EditPostPagePath := "internal/web/templates/editPost.html"
	switch r.Method {
	case "POST":
		cookie := helpers.SessionCookieGet(r)
		if cookie == nil {
			helpers.ErrorHandler(w, http.StatusUnauthorized, errors.New("Cookie failed in the Moderator Request Handler"))
			return
		}
		expTime, err := h.service.UserServiceInterface.ExtendSessionTimeout(cookie.Value)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, errors.New("Cookie cannot be extended in Approval of the Post"))
			return
		}
		err = helpers.SessionCookieExtend(r, w, expTime)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}
		postID := r.FormValue("postId")
		content := r.FormValue("updatedContent")
		intPostID, err := strconv.Atoi(postID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}
		// fmt.Println(intPostID, content)
		err = h.service.PostServiceInterface.UpdatePostContentByPostID(intPostID, content)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	default:
		helpers.ErrorHandler(w, http.StatusMethodNotAllowed, errors.New("Error in Moderator Request Handler"))
		return
	}
}

func (h *Handler) ShowMyPostsHandler(w http.ResponseWriter, r *http.Request) {
	historyPagePath := "internal/web/templates/myPosts.html"
	switch r.Method {
	case "GET":
		type templateData struct {
			MyPosts []*models.Post
			UserID  int
		}
		userID := r.FormValue("quserID")
		intuserID, err := strconv.Atoi(userID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}
		myPosts, err := h.service.PostServiceInterface.GetPostsByUserId(intuserID)
		for _, post := range myPosts {
			post.CreatedTimeString = post.CreatedTime.Format("Jan 2, 2006 at 15:04")
		}
		data := templateData{
			MyPosts: myPosts,
			UserID:  intuserID,
		}
		helpers.RenderTemplate(w, historyPagePath, data)
		return
	default:
		helpers.ErrorHandler(w, http.StatusMethodNotAllowed, errors.New("Error in Moderator Request Handler"))
		return
	}
}

func (h *Handler) ShowMyReactedPostsHandler(w http.ResponseWriter, r *http.Request) {
	historyPagePath := "internal/web/templates/myReactedPosts.html"
	switch r.Method {
	case "GET":
		type templateData struct {
			ReactedPosts []*models.Post
			UserID       int
		}
		userID := r.FormValue("quserID")
		intuserID, err := strconv.Atoi(userID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}
		var reactedPosts []*models.Post
		mapa, err := h.service.PostServiceInterface.GetMyReactedPosts(intuserID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}
		for postID, reaction := range mapa {
			post, err := h.service.PostServiceInterface.GetPostByID(postID)
			if err != nil {
				helpers.ErrorHandler(w, http.StatusInternalServerError, err)
				return
			}
			if reaction == 1 {
				post.Reaction = "Like"
			} else if reaction == -1 {
				post.Reaction = "Dislike"
			}
			reactedPosts = append(reactedPosts, post)

		}
		for _, post := range reactedPosts {
			post.CreatedTimeString = post.CreatedTime.Format("Jan 2, 2006 at 15:04")
		}
		data := templateData{
			ReactedPosts: reactedPosts,
			UserID:       intuserID,
		}
		helpers.RenderTemplate(w, historyPagePath, data)
		return
	default:
		helpers.ErrorHandler(w, http.StatusMethodNotAllowed, errors.New("Error in Moderator Request Handler"))
		return
	}
}

func (h *Handler) ShowMyReactedCommentsHandler(w http.ResponseWriter, r *http.Request) {
	historyPagePath := "internal/web/templates/myReactedComments.html"
	switch r.Method {
	case "GET":
		type templateData struct {
			ReactedComments []*models.Comment
			UserID          int
		}
		userID := r.FormValue("quserID")
		intuserID, err := strconv.Atoi(userID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}
		var reactedComments []*models.Comment
		mapa, err := h.service.CommentServiceInterface.GetMyReactedComments(intuserID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}
		for commentID, reaction := range mapa {
			comment, err := h.service.CommentServiceInterface.GetCommentByID(commentID)
			if err != nil {
				helpers.ErrorHandler(w, http.StatusInternalServerError, err)
				return
			}
			if reaction == 1 {
				comment.Reaction = "Like"
			} else if reaction == -1 {
				comment.Reaction = "Dislike"
			}
			reactedComments = append(reactedComments, comment)

		}
		for _, post := range reactedComments {
			post.CreatedTimeString = post.CreatedTime.Format("Jan 2, 2006 at 15:04")
		}
		data := templateData{
			ReactedComments: reactedComments,
			UserID:          intuserID,
		}
		helpers.RenderTemplate(w, historyPagePath, data)
		return
	default:
		helpers.ErrorHandler(w, http.StatusMethodNotAllowed, errors.New("Error in Moderator Request Handler"))
		return
	}
}

func (h *Handler) ShowMyCommentsWithPostsHandler(w http.ResponseWriter, r *http.Request) {
	path := "internal/web/templates/myCommentsWithPosts.html"
	switch r.Method {
	case "GET":
		type templateData struct {
			MyCommentedPosts []*models.CommentsWithPosts
			UserID           int
		}
		var MyCommentedPosts []*models.CommentsWithPosts
		userID := r.FormValue("quserID")
		intuserID, err := strconv.Atoi(userID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}

		// Get all comments by the user
		comments, err := h.service.CommentServiceInterface.GetCommentByUserID(intuserID)
		if err != nil {
			helpers.ErrorHandler(w, http.StatusInternalServerError, err)
			return
		}

		// For each comment, get the associated post details
		for _, comment := range comments {
			post, err := h.service.PostServiceInterface.GetPostByID(comment.PostID)
			if err != nil {
				continue // Skip if post not found or error
			}

			commentWithPost := &models.CommentsWithPosts{
				PostID:            post.PostID,
				PostTitle:         post.Title,
				PostContent:       post.Content,
				PostTimeString:    post.CreatedTime.Format("Jan 2, 2006 at 15:04"),
				CommentID:         comment.CommentID,
				CommentContent:    comment.Content,
				CommentTimeString: comment.CreatedTime.Format("Jan 2, 2006 at 15:04"),
			}
			MyCommentedPosts = append(MyCommentedPosts, commentWithPost)
		}

		data := templateData{
			MyCommentedPosts: MyCommentedPosts,
			UserID:           intuserID,
		}
		helpers.RenderTemplate(w, path, data)
		return
	default:
		helpers.ErrorHandler(w, http.StatusMethodNotAllowed, errors.New("Error in Comment Handler"))
		return
	}
}

func (h *Handler) ShowMyNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.FormValue("quserID")
	intuserID, err := strconv.Atoi(userID)
	if err != nil {
		helpers.ErrorHandler(w, http.StatusBadRequest, err)
		return
	}

	// Get reactions
	reactions, err := h.service.PostServiceInterface.GetAllMyPostsLikedByOtherUsers(intuserID)
	if err != nil {
		log.Printf("Error getting reactions: %v", err)
		reactions = []*models.PostVotes{}
	}

	// Process reactions
	for _, pv := range reactions {
		if pv.Reaction == 1 {
			pv.ReactionStr = "liked"
		} else if pv.Reaction == -1 {
			pv.ReactionStr = "disliked"
		}
		// Get post title and username
		post, _ := h.service.PostServiceInterface.GetPostByID(pv.PostID)
		if post != nil {
			pv.PostTitle = post.Title
		}
		user, _ := h.service.UserServiceInterface.GetUserByUserID(pv.UserID)
		if user != nil {
			pv.ReactorUsername = user.Username
		}
	}

	// Get comments
	comments, err := h.service.PostServiceInterface.GetAllMyPostsCommentedByOtherUsers(intuserID)
	if err != nil {
		log.Printf("Error getting comments: %v", err)
		comments = []*models.PostVotes{}
	}

	// Process comments
	for _, pv := range comments {
		post, _ := h.service.PostServiceInterface.GetPostByID(pv.PostID)
		if post != nil {
			pv.PostTitle = post.Title
		}
		user, _ := h.service.UserServiceInterface.GetUserByUserID(pv.UserID)
		if user != nil {
			pv.ReactorUsername = user.Username
		}
	}

	allNotifications := append([]*models.PostVotes{}, reactions...)
	allNotifications = append(allNotifications, comments...)
	sort.Slice(allNotifications, func(i, j int) bool {
		return allNotifications[i].Time.After(allNotifications[j].Time)
	})

	data := struct {
		Notifications []*models.PostVotes
	}{
		Notifications: allNotifications,
	}

	helpers.RenderTemplate(w, "internal/web/templates/notifications.html", data)
}

func (h *Handler) MarkNotificationSeenHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to mark notification as seen")

	var req struct {
		NotificationID int `json:"notification_id"`
	}

	// Log the raw request body
	body, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(body)) // Replace the body for later use
	log.Printf("Request body: %s", string(body))

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	log.Printf("Notification ID: %d", req.NotificationID)

	err := h.service.PostServiceInterface.MarkNotificationAsSeen(req.NotificationID)
	if err != nil {
		log.Printf("Error marking notification as seen: %v", err)
		http.Error(w, "Error marking notification as seen", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *Handler) CheckNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userID")
	intUserID, err := strconv.Atoi(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	count, err := h.service.PostServiceInterface.CountUnseenNotifications(intUserID)
	if err != nil {
		http.Error(w, "Error checking notifications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": count})
}
