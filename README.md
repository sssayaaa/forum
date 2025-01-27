# Forum 01 project

### Forum authentification

### Forum image upload

### Forum advanced features

### Forum moderation

### Forum security

For communication within the forum:

- Only registered users can create posts and comments.
- Users can associate posts with one or more categories of their choice.
- Posts and comments will be visible to all users (registered or not).
- Non-registered users can only view posts and comments.

Run MakeFile

```Text
make run
```

Check the Container

```CMD/Terminal
docker ps -a
```

Stop the container

```CMD/Terminal
make stop
make remove
```

### Docker usage

Building the Docker

```CMD/Terminal
docker build -t forum .
```

Run Docker

```CMD/Terminal
docker run --name=forum -p 8080:8080 --rm forum
```

Check the Container

```CMD/Terminal
docker ps -a
```

Stop the container

```CMD/Terminal
docker stop forum

### Usage

Cloning repository

```CMD/Terminal
git clone git@github.com:sssayaaa/forum.git
```

Go to the downloaded repository:

```CMD/Terminal
cd forum
```

Run a Server:

```CMD/Terminal
go run cmd/main.go
```

Follow the link on the terminal:

```CMD/Terminal
https://localhost:8080
```

you can play with the page

Admin account information:

```CMD/Terminal
username: admin@gmail.com
password: admin
```

# Authors:

dabduali & ssainova

