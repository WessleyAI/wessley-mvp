// Package reddit provides a scraper for Reddit's public JSON API.
package reddit

import "time"

// Post represents a scraped Reddit post with its comments.
type Post struct {
	ID          string    `json:"id"`
	Subreddit   string    `json:"subreddit"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	SelfText    string    `json:"self_text"`
	URL         string    `json:"url"`
	Permalink   string    `json:"permalink"`
	Score       int       `json:"score"`
	NumComments int       `json:"num_comments"`
	CreatedUTC  time.Time `json:"created_utc"`
	Flair       string    `json:"flair,omitempty"`
	Comments    []Comment `json:"comments"`
	ScrapedAt   time.Time `json:"scraped_at"`
}

// Comment represents a single comment on a post.
type Comment struct {
	ID         string    `json:"id"`
	Author     string    `json:"author"`
	Body       string    `json:"body"`
	Score      int       `json:"score"`
	CreatedUTC time.Time `json:"created_utc"`
	ParentID   string    `json:"parent_id"`
	Depth      int       `json:"depth"`
}

// Config controls scraper behavior.
type Config struct {
	Subreddits      []string
	PostsPerSub     int
	CommentsPerPost int
	RateLimit       time.Duration
}
