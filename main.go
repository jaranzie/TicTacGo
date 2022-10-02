package main

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"math/rand"
	"net/url"
	"os/exec"
	"time"
)

/* hack for TLS */

/**/

var gameIndex int32 = 1

type RegisteredUser struct {
	Username   string  `bson:"username" json:"username"`
	Email      string  `bson:"email" json:"email"`
	Password   string  `bson:"password" json:"password"`
	Games      []int32 `bson:"games"`
	RecentGame int32   `bson:"recentgame"`
}

type UnregisteredUser struct {
	Username         string `bson:"username" json:"username"`
	Email            string `bson:"email" json:"email"`
	Password         string `bson:"password" json:"password"`
	VerificationCode string `bson:"verificationcode"`
}

type Game struct {
	Id        int32    `bson:"_id" json:"id"`
	Grid      []string `bson:"grid" json:"grid"`
	Winner    string   `bson:"winner" json:"winner"`
	StartDate string   `bson:"start_date" json:"start_date"`
}

type Move struct {
	Move *int `json:"move"`
}

func sendEmail(user *UnregisteredUser) error {
	//msg := []byte(fmt.Sprintf("jaranzie.cse356.compas.cs.stonybrook.edu/verify?email=%s&key=%s", user.Email, user.VerificationCode)) -a "FROM:test@group1.cse356.compas.cs.stonybrook.edu"
	cmd := fmt.Sprintf("echo \"http://group1.cse356.compas.cs.stonybrook.edu/verify?email=%s&key=%s\" | mail --content-type 'text/plain; charset=ascii' -s \"Verify\" --encoding=quoted-printable %s", url.QueryEscape(user.Email), url.QueryEscape(user.VerificationCode), user.Email)
	//("127.0.0.1:25", "test@jaranzie.cse356.compas.cs.stonybrook.edu", []string{user.Email}, msg)
	_, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}
	return nil
}

func randomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:length]
}

func playGame(g *Game) {
	var winner string
	winner = checkWinner(g)
	if winner != " " {
		g.Winner = winner
		return
	}
	start := 4
	for true {
		if g.Grid[start%9] == " " {
			g.Grid[start%9] = "O"
			break
		}
		start = start + 2
	}
	winner = checkWinner(g)
	if winner != " " {
		g.Winner = winner
	}
}

func checkWinner(g *Game) string {
	grid := g.Grid
	if grid[0] != " " && grid[0] == grid[1] && grid[1] == grid[2] {
		return grid[0]
	}
	if grid[3] != " " && grid[3] == grid[4] && grid[4] == grid[5] {
		return grid[3]
	}
	if grid[6] != " " && grid[6] == grid[7] && grid[7] == grid[8] {
		return grid[6]
	}
	// Vertical
	if grid[0] != " " && grid[0] == grid[3] && grid[3] == grid[6] {
		return grid[0]
	}
	if grid[1] != " " && grid[1] == grid[4] && grid[4] == grid[7] {
		return grid[1]
	}
	if grid[2] != " " && grid[2] == grid[5] && grid[5] == grid[8] {
		return grid[2]
	}
	// Diagonal
	if grid[0] != " " && grid[0] == grid[4] && grid[4] == grid[8] {
		return grid[0]
	}
	if grid[2] != " " && grid[2] == grid[4] && grid[4] == grid[6] {
		return grid[2]
	}
	numSpaces := 0
	for i := 0; i < 9; i++ {
		if grid[i] == " " {
			numSpaces++
		}
	}
	if numSpaces == 0 {
		return "T"
	}
	return " "
}

/* Returns Inserted ID */
func newGame(gameCollection *mongo.Collection) int32 {
	var g Game
	g.Grid = []string{" ", " ", " ", " ", " ", " ", " ", " ", " "}
	g.Winner = " "
	g.StartDate = time.Now().String()
	g.Id = gameIndex
	gameIndex++
	one, err := gameCollection.InsertOne(context.TODO(), g)
	if err != nil {
		fmt.Println("Not inserted")
		return -1
	}
	return one.InsertedID.(int32)
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
		res := unregisteredusers.FindOne(context.TODO(), bson.D{{`email`, newUser.Email}})
		if res.Err() != mongo.ErrNoDocuments {
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		res = unregisteredusers.FindOne(context.TODO(), bson.D{{`username`, newUser.Username}})
		if res.Err() != mongo.ErrNoDocuments {
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		res = registeredusers.FindOne(context.TODO(), bson.D{{`username`, newUser.Username}})
		if res.Err() != mongo.ErrNoDocuments {
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		res = registeredusers.FindOne(context.TODO(), bson.D{{`email`, newUser.Email}})
		if res.Err() != mongo.ErrNoDocuments {
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		if len(newUser.Username) == 0 || len(newUser.Email) == 0 || len(newUser.Password) == 0 {
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		newUser.VerificationCode = randomString(10)
		err := sendEmail(&newUser)
		if err != nil {
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		_, err = unregisteredusers.InsertOne(context.TODO(), newUser)
		//Check errors
		return c.JSON(fiber.Map{"status": "OK"})
	})

	/*app.Get("/verify", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "OK"})
	})*/

	app.Get("/verify", func(c *fiber.Ctx) error {
		email, _ := url.PathUnescape(c.Query("email"))
		key, _ := url.PathUnescape(c.Query("key"))
		var u UnregisteredUser
		res := unregisteredusers.FindOne(context.TODO(), bson.D{{`email`, email}})
		if res.Err() == mongo.ErrNoDocuments {
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		err := res.Decode(&u)
		if err != nil {
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		if u.VerificationCode == key {
			r := RegisteredUser{u.Username, u.Email, u.Password, []int32{}, -1}
			_, err := registeredusers.InsertOne(context.TODO(), r)
			if err != nil {
				return c.JSON(fiber.Map{"status": "ERROR"})
			}
			_, err = unregisteredusers.DeleteOne(context.TODO(), bson.M{"email": email})
			if err != nil {
				return c.JSON(fiber.Map{"status": "ERROR"})
			}
			return c.JSON(fiber.Map{"status": "OK"})
		}
		return c.JSON(fiber.Map{"status": "ERROR"})
	})

	app.Post("/login", func(c *fiber.Ctx) error {
		var user RegisteredUser
		var authUser RegisteredUser
		_ = c.BodyParser(&user)
		err := registeredusers.FindOne(context.TODO(), bson.D{{`username`, user.Username}}).Decode(&authUser)
		if err != nil || user.Password != authUser.Password {
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		sess, _ := store.Get(c)
		sess.Set("username", authUser.Username)
		err = sess.Save()
		if err != nil {
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		return c.JSON(fiber.Map{"status": "OK"})
	})

	app.Post("/logout", func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return fiber.ErrBadRequest
		}
		_ = sess.Destroy()
		err = sess.Save()
		if err != nil {
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		return c.JSON(fiber.Map{"status": "OK"})
	})

	app.Post("/ttt/play", func(c *fiber.Ctx) error {
		var mo Move
		if err := c.BodyParser(&mo); err != nil {
			fmt.Println("bad")
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		sess, err := store.Get(c)
		if err != nil {
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		username := sess.Get("username")
		if username == nil {
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		var user RegisteredUser
		err = registeredusers.FindOne(context.TODO(), bson.D{{`username`, username}}).Decode(&user)
		var game Game
		if user.RecentGame == -1 {
			newGameID := newGame(games)
			_, _ = registeredusers.UpdateOne(context.TODO(), bson.D{{"username", user.Username}}, bson.D{{"$set", bson.D{{"recentgame", newGameID}}}})
			err = games.FindOne(context.TODO(), bson.D{{"_id", newGameID}}).Decode(&game)
			user.RecentGame = newGameID
		} else {
			err = games.FindOne(context.TODO(), bson.D{{"_id", user.RecentGame}}).Decode(&game)
		}
		if mo.Move == nil {
			_, err = games.UpdateOne(context.TODO(), bson.D{{"_id", user.RecentGame}}, bson.D{{"$set", bson.D{{"grid", game.Grid}}}})
			return c.JSON(fiber.Map{"status": "OK", "grid": game.Grid, "winner": game.Winner})
		}
		grid := &game.Grid
		if &mo.Move == nil {
			return c.JSON(user.RecentGame)
		}

		if (*grid)[int(*mo.Move)] != " " {
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		(*grid)[*mo.Move] = "X"
		/*
			Playing Game
		*/
		/**/

		playGame(&game)
		if game.Winner != " " { /* newGame(games) instead of -1 maybe */
			_, err = games.UpdateOne(context.TODO(), bson.D{{"_id", user.RecentGame}}, bson.D{{"$set", bson.D{{"grid", game.Grid}}}})
			_, err = games.UpdateOne(context.TODO(), bson.D{{"_id", user.RecentGame}}, bson.D{{"$set", bson.D{{"winner", game.Winner}}}})
			update := bson.D{{"$set", bson.D{{"recentgame", -1}}}, {"$push", bson.D{{"games", user.RecentGame}}}}
			_, _ = registeredusers.UpdateOne(context.TODO(), bson.D{{"username", user.Username}}, update)
			return c.JSON(fiber.Map{"status": "OK", "grid": game.Grid, "winner": game.Winner})
		} else {
			_, err := games.UpdateOne(context.TODO(), bson.D{{"_id", user.RecentGame}}, bson.D{{"$set", bson.D{{"grid", game.Grid}}}})
			if err != nil {
				return err
			}
			fmt.Println(game)
			return c.JSON(fiber.Map{"status": "OK", "grid": game.Grid, "winner": game.Winner})
		}
	})

	app.Post("/listgames", func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		var user RegisteredUser
		err = registeredusers.FindOne(context.TODO(), bson.D{{`username`, sess.Get("username")}}).Decode(&user)
		if err != nil {
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		var game Game
		type listGameSt struct {
			Id        int32  `json:"id"`
			StartDate string `json:"start_date"`
		}
		type listGameRet struct {
			Status string       `json:"status"`
			Games  []listGameSt `json:"games"`
		}
		var length int
		if user.RecentGame == -1 {
			length = len(user.Games)

		} else {
			length = len(user.Games) + 1
		}
		gameArray := make([]listGameSt, length)
		for i := 0; i < len(user.Games); i++ {
			err := games.FindOne(context.TODO(), bson.D{{"_id", user.Games[i]}}).Decode(&game)
			gameArray[i] = listGameSt{game.Id, game.StartDate}
			if err != nil {
				return err
			}
		}
		if user.RecentGame != -1 {
			err = games.FindOne(context.TODO(), bson.D{{"_id", user.RecentGame}}).Decode(&game)
			gameArray[len(user.Games)] = listGameSt{game.Id, game.StartDate}
			if err != nil {
				return err
			}
		}
		return c.JSON(listGameRet{"OK", gameArray})
	})

	app.Post("/getgame", func(c *fiber.Ctx) error {
		var game Game
		if err := c.BodyParser(&game); err != nil {
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		err = games.FindOne(context.TODO(), bson.D{{"_id", game.Id}}).Decode(&game)
		if err != nil {
			return c.JSON(fiber.Map{"status": "ERROR"})
		}

		type GetGameResponse struct {
			Status string   `json:"status"`
			Grid   []string `json:"grid"`
			Winner string   `json:"winner"`
		}
		//return c.JSON(fiber.Map{"status": "OK", "grid": string(game.Grid), "winner": game.Winner})
		return c.JSON(GetGameResponse{"OK", game.Grid, game.Winner})
		//return c.JSON(fmt.Sprintf(`"status":"OK", "grid" : "%s", "winner": '%s'`, string(game.Grid), string(game.Winner)))
	})

	app.Post("/getscore", func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		var user RegisteredUser
		err = registeredusers.FindOne(context.TODO(), bson.D{{`username`, sess.Get("username")}}).Decode(&user)
		if err != nil {
			fmt.Println("prob1")
			return c.JSON(fiber.Map{"status": "ERROR"})
		}
		wins := 0
		ties := 0
		wopr := 0
		gs := user.Games

		var game Game
		for i := 0; i < len(gs); i++ {
			err := games.FindOne(context.TODO(), bson.D{{"_id", gs[i]}}).Decode(&game)
			if err != nil {

				return c.JSON(fiber.Map{"status": "ERROR"})
			}
			if game.Winner == "X" {
				wins++
				continue
			} else if game.Winner == "T" {
				ties++
				continue
			} else if game.Winner == "O" {
				wopr++
				continue
			}
		}
		fmt.Println(wins)
		fmt.Println(wopr)
		fmt.Println(ties)
		return c.JSON(fiber.Map{"status": "OK", "human": wins, "wopr": wopr, "tie": ties})
	})

	FiberErr := app.Listen(":80")
	if FiberErr != nil {
		return
	}
}
