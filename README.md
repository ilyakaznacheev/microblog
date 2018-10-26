# Simple GraphQL blog microservice

This is the simple service able to create, read, update and delete simple text blogs.
Service is based on [gramework](https://github.com/gramework/gramework) and uses following tools:

- GraphQL
- Redis
- PostgreSQL

## Starting

To start server please adjust ```config.json``` and run package:
```bash
go run *.go
```
And then your GraphQL server will be available at the address from ```config.json``` (default ```localhost:8080```).

## Service provides following actions:

Queries:
- get single post
- get post list

Mutations:
- create post
- update post
- delete post

## Usage example

You can send requests any way you want, for example
```bash
curl -XPOST -d '{"query": "{ getPost(id:1) { id, description, title } }"}' -H "Content-Type: application/json" localhost:8080/graphql
```

## Query examples

### Get single post

```graphql
{ getPost(id:3)
	{ 
		id, 
		description, 
		title, 
		shortText, 
		fullText, 
		URI,
		readCount
	} 
}
```

### Get post list

```graphql
{ getPosts 
	{ 
		id, 
		description, 
		title, 
		shortText, 
		fullText, 
		URI,
		readCount
	} 
}
```

### Create post

```graphql
mutation {
	createPost(post:{
        title: "Post title",
        description: "Some info",
        shortText: "Text",
        fullText: "More text",
        URI: "http://some/usefull/link",
    })
	}
```

### Update post
```graphql
mutation {
  updatePost(id:3, post: {
      title: "Some info", 
      description: "bbq3456", 
      shortText: "Text", 
      fullText: "More text", 
      URI: "http://another/usefull/link"
      }) {
    id
    title
    description
    shortText
    fullText
    URI
    readCount
  }
}
```

### Delete post
```graphql
mutation {
	deletePost(id: 3)
}
```