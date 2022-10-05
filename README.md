# TicTacGo
Warmup Project 2 for CSE 356

Decided to try and build API in Go. First experience using it. Only a back end API

Requirments:

1. Develop a user-creation system validated with email. Handle duplicate credentials.
/adduser { username, password, email } -- Creates a disabled user that cannot log in.
GET /verify { email, key } -- Verification link with the two parameters in the query string is sent by email. Do not use a third-party mail service (i.e. gmail) for your mail server.

2. Add cookie-based session support. Ideally, make sessions persist across server restarts.
/login { username, password }
/logout { }

3. Modify your Tic-Tac-Toe REST service at http://yourserver/ttt/play to take as input a JSON object including a 'move' property to indicate on which square (0-indexed, in row-major order) the human (X) is making a move in the current game. The game state should be saved even after refreshing. The server should respond with a JSON object that includes a 'grid' property and a 'winner' property as in WP#1. Making a request with { move: null } should return the current grid without making a move. Once a winning or tying move has been sent to the server, the server should consider the game completed and reset the grid. 

4. Maintain the history of previously played games by the user that is currently logged in:
/listgames { }
  { status: 'OK', games: [{id, start_date}, …] }
/getgame { id }
  { status: 'OK', grid: ['X','O', …], winner: 'X' }
/getscore { }
  { status: 'OK', human: 0, wopr: 5, tie: 10 }.

Clarification: all of the above API calls must be to POST routes unless otherwise specified. Add a 'status' property to all JSON responses with the value 'OK' or 'ERROR'. Use your judgement for possible operations/situations that may lead to an error.

All POST request responses must also contain the header field X-CSE356 with the value containing the ID copied from the course interface. 

To use, install postfix, gnumailutils and Golang.
