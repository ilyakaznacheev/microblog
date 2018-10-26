package main

import (
	"context"
	"log"
	"strconv"

	graphql "github.com/graph-gophers/graphql-go"
)

type postInput struct {
	ID          int32
	Title       string
	Description string
	ShortText   string
	FullText    string
	URI         string
}

// Resolver is the main GraphQL resolver
type Resolver struct {
	rh *repoHandler
}

// GetPost resolves getPost query
func (r *Resolver) GetPost(ctx context.Context, args struct{ ID int32 }) (*PostRepr, error) {
	log.Println("GetPost")
	post, err := r.rh.getPost(args.ID)
	return post, err
}

// GetPosts resolves getPosts query
func (r *Resolver) GetPosts(ctx context.Context) (*[]*PostRepr, error) {
	log.Println("GetPosts")
	posts, err := r.rh.getPostList()
	return &posts, err
}

// CreatePost createPost mutation
func (r *Resolver) CreatePost(ctx context.Context, args struct{ Post postInput }) (
	*graphql.ID, error,
) {
	log.Println("CreatePost")
	id, err := r.rh.createPost(args.Post)
	value := graphql.ID(strconv.Itoa(int(id)))
	return &value, err
}

// UpdatePost updatePost mutation
func (r *Resolver) UpdatePost(ctx context.Context, args struct {
	ID   int32
	Post postInput
}) (*PostRepr, error) {
	log.Println("UpdatePost")
	post, err := r.rh.updatePost(args.ID, args.Post)
	return post, err
}

// DeletePost deletePost mutation
func (r *Resolver) DeletePost(ctx context.Context, args struct{ ID int32 }) (*bool, error) {
	log.Println("DeletePost")
	err := r.rh.deletePost(args.ID)
	res := (err == nil)
	return &res, err
}

func newResolver(rh *repoHandler) *Resolver {
	return &Resolver{rh: rh}
}
