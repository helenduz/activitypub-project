# Simple ActivityPub Server in Go
This is a simple ActivityPub HTTP API server that supports a very small subset of the Social API (client to server protocol) and Federation API (server to server protocol) detailed in the [W3C Specification](https://www.w3.org/TR/activitypub). In particular, the following functionalities are supported:

* Creating users on the server that are discoverable on the web via the WebFinger protocol;
* Sending messages to all followers' inboxes;
* Receiving Follow requests from other servers and responding with Accept;
* Storing all created users and sent messages in a database.


In short, once you start running the server on a domain, you can interact with it using the browser or other HTTP clients like Postman. After a user is created, you will receive an API key in the response, which you will attach to your message-sending requests. Your user can be discovered by other servers that implement the ActivityPub protocol, and users on these servers (e.g. Mastodon) can follow you. Once you have followers, you can send messages to their inboxes, which will show up in their timelines.

This repo is built upon this reference implementation in Express: https://github.com/dariusk/express-activitypub. Also, I adopted and modified the admin page HTML in this reference repo for my own use.

## Install and Run
Clone the repository, `cd` into its root directory, then `go install` to install all dependencies. 
Add a `.env` file in the root directory, with the following information:
```
PORT=port_to_run_on
ADMIN_USER=pick_a_username
ADMIN_PASS=pick_a_password
DOMAIN=domain_you_own
```
Run the server with `make`, or `go build`, or any other methods you like (tip: use [Air](https://github.com/cosmtrek/air) if you want your server to automatically rebuild and restart on file changes). 

If you are running the server locally (in which case you will only be able to test the account creation functionality), you can pick anything for DOMAIN. If you are testing using reverse proxies like [ngrok](https://ngrok.com/), what you need to do is to (1) install ngrok (2) run `ngrok http 3000` (if you run your server on port 3000), which will give you a testing domain (3) update your `.env` file and make DOMAIN the testing domain you get from ngrok (4) restart your server. 

Note that the database gets erased when you restart your server.

## Repo Structure
<img width="180" alt="Screenshot 2024-01-20 at 4 05 06 PM" src="https://github.com/helenduz/activitypub-project/assets/62923883/6ceb133d-23c4-454b-914f-abcb0e93c34f">

The main server file is `main.go`, which sets up the database and routers, and initializes the server to listen on a port. We use a SQLite database, which stores all data inside `ap-server.db`. Route handlers sit inside the `pkg/handlers` directory. In particular, the routes and their handler files are as follows:

* `/admin`, a route that returns the static HTML file for the admin page
* `/.well-known/webfinger`, routes that respond to requests for discovering users on our server via the WebFinger protocol; handlers live in `pkg/handlers/webfinger.go`
* `/u/{name}` and `/u/{name}/followers`, routes that serves JSON data, which allow other servers to get information about the user and get its followers; handlers live in `pkg/handlers/user.go`
* `/api/admin/create`, a route that handles creating a new account (along with its public-private key pair, API key, WebFinger record, etc.) and adding it to our database; handlers live in `pkg/handlers/admin.go`
* `/api/inbox`, a route that can receive messages from other servers (currently it can only handle Follow objects and respond with Accept objects); handlers live in `pkg/handlers/inbox.go`
* `/api/send`, a route that wraps the given text inside a Note object and sends the Create object of that note to all followers' inboxes (which will then appear on their timelines); handlers live in `pkg/handlers/send.go`

In addition, `pkg/middlewares` contains helper functions for a basic HTTP authorizer used by the route `/api/admin/create`; `pkg/utils` contains helper functions for generating encryption keys; and `pkg/app` contains server states and resources (such as the domain and database connector). 

It is also worth pointing out that `/api/send`, `/api/admin/create`, and `/admin` are routes that are specific to our server (in that they are used only by clients that wish to interact with our server), while `/u/{name}`, `/u/{followers}`, `/api/inbox`, and `/.well-known/webfinger` are routes that will be visited by other ActivityPub servers, therefore their naming in fact follows the ActivityPub convention.

