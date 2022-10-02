package main

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"math/rand"
	"time"
)

var mongoURL string = ""

type RegisteredUser struct {
	Username   string               `bson:"username" json:"username"`
	Email      string               `bson:"email" json:"email"`
	Password   string               `bson:"password" json:"password"`
	Games      []primitive.ObjectID `bson:"games"`
	RecentGame Game                 `RecentGame:"recentgame"`
}

type UnregisteredUser struct {
	Username         string `bson:"username" json:"username"`
	Email            string `bson:"email" json:"email"`
	Password         string `bson:"password" json:"password"`
	VerificationCode string `bson:"verificationcode"`
}

type Game struct {
	Grid   []rune `bson:"grid" json:"grid"`
	Winner rune   `bson:"winner" json:"winner"`
}

type Move struct {
	Move int `json:"move"`
}

func randomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:length]
}

func newGame() Game {
	var g Game
	g.Grid = []rune{' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' '}
	g.Winner = ' '
	return g
}

func playGame(g *Game) {
	var winner rune
	winner = checkWinner(g)
	if winner != ' ' {
		g.Winner = winner
		return
	}
	start := 4
	for true {
		if g.Grid[start] == ' ' {
			g.Grid[start] = 'O'
			break
		}
		start = start + 2
	}
	winner = checkWinner(g)
	if winner != ' ' {
		g.Winner = winner
	}
}

func checkWinner(g *Game) rune {
	grid := g.Grid
	if grid[0] != ' ' && grid[0] == grid[1] && grid[1] == grid[2] {
		return grid[0]
	}
	if grid[3] != ' ' && grid[3] == grid[4] && grid[4] == grid[5] {
		return grid[3]
	}
	if grid[6] != ' ' && grid[6] == grid[7] && grid[7] == grid[8] {
		return grid[6]
	}
	// Vertical
	if grid[0] != ' ' && grid[0] == grid[3] && grid[3] == grid[6] {
		return grid[0]
	}
	if grid[1] != ' ' && grid[1] == grid[4] && grid[4] == grid[7] {
		return grid[1]
	}
	if grid[2] != ' ' && grid[2] == grid[5] && grid[5] == grid[8] {
		return grid[2]
	}
	// Diagonal
	if grid[0] != ' ' && grid[0] == grid[4] && grid[4] == grid[8] {
		return grid[0]
	}
	if grid[2] != ' ' && grid[2] == grid[4] && grid[4] == grid[6] {
		return grid[2]
	}
	numSpaces := 0
	for i := 0; i < 9; i++ {
		if grid[i] == ' ' {
			numSpaces++
		}
	}
	if numSpaces == 0 {
		return 'T'
	}
	return ' '
}

func main() {
	/*
		Connecting to Mongo
	*/
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
	registeredusers := client.Database("warmup2").Collection("RegisteredUser")
	unregisteredusers := client.Database("warmup2").Collection("UnregisteredUser")
	games := client.Database("warmup2").Collection("Games")
	//activeGames := client.Database("warmup2").Collection("ActiveGames")

	app := fiber.New()
	store := session.New()
	app.Use(func(c *fiber.Ctx) error {
		// Set some security headers:
		c.Set("X-CSE356", "6306bbbd2988c22186b26cb2")
		// Go to next middleware:
		return c.Next()
	})

	app.Post("/adduser", func(c *fiber.Ctx) error {
		var newUser UnregisteredUser
		if err := c.BodyParser(&newUser); err != nil {
			return err
		}
		if len(newUser.Username) == 0 || len(newUser.Email) == 0 || len(newUser.Password) == 0 {
			return err
		}
		newUser.VerificationCode = randomString(10)
		go func() {
			//SEND MAIL "url/verify?email=newUser.email&key=newUser.verificationCode
			_, err = unregisteredusers.InsertOne(context.TODO(), newUser)
		}()
		//Check errors
		return c.SendString("User Creation Success")
	})

	app.Get("/verify", func(c *fiber.Ctx) error {
		email := c.Query("email")
		key := c.Query("key")
		var u UnregisteredUser
		err := unregisteredusers.FindOne(context.TODO(), bson.D{{`email`, email}}).Decode(&u)
		if err != nil {
			return fiber.ErrBadRequest
		}
		if u.VerificationCode == key {
			r := RegisteredUser{u.Username, u.Email, u.Password, []primitive.ObjectID{}, newGame()}
			_, err = registeredusers.InsertOne(context.TODO(), r)
			if err != nil {
				return fiber.ErrBadRequest
			}
			_, err = unregisteredusers.DeleteOne(context.TODO(), bson.D{{`email`, email}})
			if err != nil {
				return fiber.ErrBadRequest
			}
			return c.SendString("Verified")
		}
		return fiber.ErrBadRequest
	})

	app.Post("/login", func(c *fiber.Ctx) error {
		var user RegisteredUser
		var authUser RegisteredUser
		_ = c.BodyParser(&user)
		err := registeredusers.FindOne(context.TODO(), bson.D{{`username`, user.Username}}).Decode(&authUser)
		if err != nil || user.Password != authUser.Password {
			return fiber.ErrBadRequest
		}
		sess, _ := store.Get(c)
		sess.Set("username", authUser.Username)
		return c.SendString("Login Succesful")
	})

	app.Post("/logout", func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return fiber.ErrBadRequest
		}
		_ = sess.Destroy()
		return c.SendString("Logout Succesful")
	})

	app.Post("/ttt/play", func(c *fiber.Ctx) error {
		var mo Move
		if err := c.BodyParser(mo); err != nil {
			return err
		}
		sess, err := store.Get(c)
		if err != nil {
			return c.SendString("Not logged in")
		}
		var user RegisteredUser
		err = registeredusers.FindOne(context.TODO(), bson.D{{`username`, sess.Get("username")}}).Decode(&user)
		grid := user.RecentGame.Grid
		if &mo.Move == nil {
			return c.JSON(user.RecentGame)
		}
		if grid[int(mo.Move)] != ' ' {
			return fiber.ErrConflict
		}
		grid[mo.Move] = 'X'
		playGame(&user.RecentGame)
		if user.RecentGame.Winner != ' '
	})

	FiberErr := app.Listen(":80")
	if FiberErr != nil {
		return
	}
}
