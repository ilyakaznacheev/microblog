package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/go-pg/pg"
	"github.com/go-redis/redis"
	"github.com/graph-gophers/graphql-go"
)

const (
	redKey          = "microblog"
	redChangeKey    = "last-change"
	redPostKey      = "post-key"
	redPostListKey  = "post-list"
	redReadCountKey = "post-read"
)

var (
	// ErrRedisCacheVersion error while redis cache version check
	ErrRedisCacheVersion = errors.New("cache outdated")
)

// PostRepr is a post model
type PostRepr struct {
	data *Post
	rh   *repoHandler
}

func (p *PostRepr) getReadCounter() int {
	return p.rh.updateReadCounter(p.data.ID)
}

// ID returns ID field value
func (p *PostRepr) ID(ctx context.Context) *graphql.ID {
	id := graphql.ID(strconv.Itoa(int(p.data.ID)))
	return &id
}

// TITLE returns Title field value
func (p *PostRepr) TITLE(ctx context.Context) *string {
	return &p.data.Title
}

// DESCRIPTION returns Description field value
func (p *PostRepr) DESCRIPTION(ctx context.Context) *string {
	return &p.data.Description
}

// SHORTTEXT returns ShortText field value
func (p *PostRepr) SHORTTEXT(ctx context.Context) *string {
	return &p.data.ShortText
}

// FULLTEXT returns FullText field value
func (p *PostRepr) FULLTEXT(ctx context.Context) *string {
	return &p.data.FullText
}

// URI returns URI field value
func (p *PostRepr) URI(ctx context.Context) *string {
	return &p.data.URI
}

// READCOUNT returns ReadCount field value
func (p *PostRepr) READCOUNT(ctx context.Context) *int32 {
	count := int32(p.getReadCounter())
	return &count
}

// Post represents DB structure of post entity
type Post struct {
	ID          int32  `sql:"id,pk"`
	Title       string `sql:"title"`
	Description string `sql:"description"`
	ShortText   string `sql:"shorttext"`
	FullText    string `sql:"fulltext"`
	URI         string `sql:"uri"`
	ReadCount   int32  `sql:"-"`
}

type dbHandler struct {
	db *pg.DB
}

func newDbHandler(conf *ConfigData) *dbHandler {
	db := pg.Connect(&pg.Options{
		User:     conf.Database.User,
		Password: conf.Database.Pass,
		Database: conf.Database.Name,
	})
	return &dbHandler{
		db: db,
	}
}

func (dh *dbHandler) shutdown() {
	dh.db.Close()
}

func (dh *dbHandler) getPost(id int32) (*Post, error) {
	post := &Post{ID: id}
	err := dh.db.Select(post)
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (dh *dbHandler) getPostList() ([]*Post, error) {
	var posts []Post
	err := dh.db.Model(&posts).Select()
	if err != nil {
		return []*Post{}, err
	}
	postRef := make([]*Post, 0, len(posts))
	for idx := range posts {
		postRef = append(postRef, &posts[idx])
	}
	return postRef, nil
}

func (dh *dbHandler) createPost(post *Post) error {
	_, err := dh.db.Model(post).Returning("id").Insert()
	if err == nil {
		log.Printf("created post %d\n", post.ID)
	} else {
		log.Println("error during post creation:", err)
	}

	return err
}

func (dh *dbHandler) updatePost(post *Post) error {
	err := dh.db.Update(post)
	if err == nil {
		log.Printf("updated post %d\n", post.ID)
	} else {
		log.Println("error during post update:", err)
	}
	return err
}

func (dh *dbHandler) deletePost(id int32) error {
	post := &Post{ID: id}
	err := dh.db.Delete(post)
	if err == nil {
		log.Printf("deleted post %d\n", post.ID)
	} else {
		log.Println("error during post deletion:", err)
	}
	return err
}

// RedisContainer is a redis main container structure
type RedisContainer struct {
	Version int
	Content string
}

type redisHandler struct {
	cl *redis.Client
}

func newRedisHandler(conf *ConfigData) *redisHandler {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Address,
		Password: config.Redis.Password,
		DB:       config.Redis.DataBase,
	})
	return &redisHandler{
		cl: client,
	}
}

func (rh *redisHandler) get(entity, key string) (string, error) {
	version, _ := rh.getChangeCounter(entity, key)
	entityKey := fmt.Sprintf("%s:%s:%s", redKey, entity, key)
	responseData, err := rh.cl.Get(entityKey).Result()
	if err != nil {
		return "", err
	}

	container := &RedisContainer{}
	json.Unmarshal([]byte(responseData), container)
	if version > container.Version {
		return "", ErrRedisCacheVersion
	}
	return container.Content, nil
}

func (rh *redisHandler) set(entity, key, requestData string, version int) error {
	entityKey := fmt.Sprintf("%s:%s:%s", redKey, entity, key)

	container := &RedisContainer{
		Version: version,
		Content: requestData,
	}
	requestJSON, err := json.Marshal(container)
	if err != nil {
		log.Panic(err)
	}
	err = rh.cl.Set(entityKey, string(requestJSON), 0).Err()
	if err != nil {
		return err
	}
	return nil
}

func (rh *redisHandler) updateChangeCounter(entity, key string) int {
	entityKey := fmt.Sprintf("%s:%s:%s:%s", redKey, redChangeKey, entity, key)
	counter, err := rh.cl.Incr(entityKey).Result()
	if err != nil {
		log.Println(err)
	}

	return int(counter)
}

func (rh *redisHandler) getChangeCounter(entity, key string) (int, error) {
	entityKey := fmt.Sprintf("%s:%s:%s:%s", redKey, redChangeKey, entity, key)
	counterStr, err := rh.cl.Get(entityKey).Result()
	if err != nil {
		return 0, err
	}
	counter, err := strconv.Atoi(counterStr)
	if err != nil {
		log.Panic(err)
	}
	return counter, nil
}

func (rh *redisHandler) updateReadCounter(entity, key string) int {
	entityKey := fmt.Sprintf("%s:ctr:%s:%s", redKey, entity, key)
	counter, err := rh.cl.Incr(entityKey).Result()
	if err != nil {
		log.Println(err)
	}

	return int(counter)
}

func (rh *redisHandler) shutdown() {
	rh.cl.Close()
}

type repoHandler struct {
	dbh *dbHandler
	rdh *redisHandler
	dmx *sync.Mutex
}

func newRepoHandler(conf *ConfigData) *repoHandler {
	return &repoHandler{
		dbh: newDbHandler(conf),
		rdh: newRedisHandler(conf),
		dmx: &sync.Mutex{},
	}
}

func (h *repoHandler) getPost(id int32) (*PostRepr, error) {
	// read from cache
	cachedData, err := h.rdh.get(redPostKey, fmt.Sprintf("%d", id))
	if err == nil {
		postCache := &Post{}
		json.Unmarshal([]byte(cachedData), postCache)

		return &PostRepr{
			data: postCache,
			rh:   h,
		}, nil
	}

	// read from db
	post, err := h.dbh.getPost(id)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// update cache
	cacheVersion := h.rdh.updateChangeCounter(redPostKey, fmt.Sprintf("%d", id))

	newCachedData, err := json.Marshal(*post)
	if err != nil {
		log.Panic(err)
	}
	err = h.rdh.set(
		redPostKey,
		fmt.Sprintf("%d", id),
		string(newCachedData),
		cacheVersion,
	)
	if err != nil {
		log.Panic(err)
	}

	return &PostRepr{
		data: post,
		rh:   h,
	}, nil
}

func (h *repoHandler) getPostList() ([]*PostRepr, error) {
	var postListCache []Post
	// read from cache
	cachedData, err := h.rdh.get(redPostListKey, "")
	if err == nil {
		postListCache = make([]Post, 0)
		json.Unmarshal([]byte(cachedData), &postListCache)
		pl := make([]*PostRepr, 0, len(postListCache))

		for idx := range postListCache {
			pr := &PostRepr{
				data: &postListCache[idx],
				rh:   h,
			}
			pl = append(pl, pr)

		}

		return pl, nil
	}

	// read from db
	posts, err := h.dbh.getPostList()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	posrRP := make([]*PostRepr, 0, len(posts))
	for _, post := range posts {
		posrRP = append(posrRP, &PostRepr{
			data: post,
			rh:   h,
		})
	}

	// update cache
	cacheVersion := h.rdh.updateChangeCounter(redPostListKey, "")

	postListCache = make([]Post, 0, len(posrRP))
	for idx := range posrRP {
		postListCache = append(postListCache, *posrRP[idx].data)
	}
	newCachedData, err := json.Marshal(&postListCache)
	if err != nil {
		log.Panic(err)
	}
	err = h.rdh.set(
		redPostListKey,
		"",
		string(newCachedData),
		cacheVersion,
	)
	if err != nil {
		log.Panic(err)
	}

	return posrRP, nil
}

func (h *repoHandler) createPost(post postInput) (int32, error) {
	newPost := &Post{
		Title:       post.Title,
		Description: post.Description,
		ShortText:   post.ShortText,
		FullText:    post.FullText,
		URI:         post.URI,
	}
	h.dmx.Lock()
	defer h.dmx.Unlock()
	err := h.dbh.createPost(newPost)
	if err == nil {
		// update cache version
		h.rdh.updateChangeCounter(redPostListKey, "")
	}
	return newPost.ID, err
}

func (h *repoHandler) updatePost(id int32, post postInput) (*PostRepr, error) {
	updPost := &Post{
		ID:          id,
		Title:       post.Title,
		Description: post.Description,
		ShortText:   post.ShortText,
		FullText:    post.FullText,
		URI:         post.URI,
	}
	h.dmx.Lock()
	defer h.dmx.Unlock()
	err := h.dbh.updatePost(updPost)
	if err == nil {
		// update cache version
		h.rdh.updateChangeCounter(redPostListKey, "")
		h.rdh.updateChangeCounter(redPostKey, fmt.Sprintf("%d", updPost.ID))
	}
	return &PostRepr{
		data: updPost,
		rh:   h,
	}, err
}

func (h *repoHandler) deletePost(id int32) error {
	h.dmx.Lock()
	defer h.dmx.Unlock()
	err := h.dbh.deletePost(id)
	if err == nil {
		// update cache version
		h.rdh.updateChangeCounter(redPostListKey, "")
		h.rdh.updateChangeCounter(redPostKey, fmt.Sprintf("%d", id))
	}
	return err
}

func (h *repoHandler) updateReadCounter(id int32) int {
	return int(h.rdh.updateReadCounter(redPostKey, fmt.Sprintf("%d", id)))
}

func (h *repoHandler) shutdown() {
	h.dbh.shutdown()
	h.rdh.shutdown()
}
