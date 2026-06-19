// Command genexamples generates a large Senda collection of real public-API
// requests (useful for exercising collection load/open performance), and seeds
// varied run history so the sidebar recency pills show a realistic spread
// (fresh / recent / stale / error) the moment the collection is opened.
//
// It reuses the production store/history packages so the on-disk YAML and
// history format always match what the app itself writes.
//
//	go run ./scripts/genexamples [DEST]
//
// DEST defaults to examples-collection/large (relative to the repo root). The
// tree is also packed into <DEST>.senda.zip — a single reproducible blob that
// is committed to git (the tree itself stays gitignored).
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"senda/internal/history"
	"senda/internal/model"
	"senda/internal/store"
)

// ---- request builders ------------------------------------------------------

var jsonHeader = []model.KV{{Key: "Content-Type", Value: "application/json", Enabled: true}}

func get(name, url string) model.Request {
	return model.Request{Name: name, Method: "GET", URL: url,
		Body: model.Body{Type: model.BodyNone}, Auth: model.Auth{Type: model.AuthInherit}}
}

func withBody(name, method, url string, body any) model.Request {
	raw, _ := json.MarshalIndent(body, "", "  ")
	return model.Request{Name: name, Method: method, URL: url, Headers: jsonHeader,
		Body: model.Body{Type: model.BodyJSON, Raw: string(raw)},
		Auth: model.Auth{Type: model.AuthInherit}}
}

func post(name, url string, body any) model.Request  { return withBody(name, "POST", url, body) }
func put(name, url string, body any) model.Request   { return withBody(name, "PUT", url, body) }
func patch(name, url string, body any) model.Request { return withBody(name, "PATCH", url, body) }
func del(name, url string) model.Request {
	return model.Request{Name: name, Method: "DELETE", URL: url,
		Body: model.Body{Type: model.BodyNone}, Auth: model.Auth{Type: model.AuthInherit}}
}

func gql(name, url, query, vars string) model.Request {
	return model.Request{Name: name, Method: "POST", URL: url,
		Body: model.Body{Type: model.BodyGraphQL, Raw: query, Variables: vars},
		Auth: model.Auth{Type: model.AuthInherit}}
}

type folder struct {
	segs []string
	reqs []model.Request
}

func rangeGet(prefix, urlFmt string, lo, hi int) []model.Request {
	var out []model.Request
	for i := lo; i < hi; i++ {
		out = append(out, get(fmt.Sprintf("%s %d", prefix, i), fmt.Sprintf(urlFmt, i)))
	}
	return out
}

func m(reqs ...model.Request) []model.Request { return reqs }

func concat(groups ...[]model.Request) []model.Request {
	var out []model.Request
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

// ---- the collection spec ---------------------------------------------------

func spec() []folder {
	const (
		jp = "https://jsonplaceholder.typicode.com"
		hb = "https://httpbin.org"
		pe = "https://postman-echo.com"
		gh = "https://api.github.com"
		pk = "https://pokeapi.co/api/v2"
		rc = "https://restcountries.com/v3.1"
		om = "https://api.open-meteo.com/v1"
		cg = "https://api.coingecko.com/api/v3"
		fr = "https://api.frankfurter.app"
		rm = "https://rickandmortyapi.com/api"
		ol = "https://openlibrary.org"
		rr = "https://reqres.in/api"
		fs = "https://fakestoreapi.com"
		dc = "https://deckofcardsapi.com/api/deck"
	)

	return []folder{
		// ---- JSONPlaceholder ----
		{[]string{"JSONPlaceholder", "Posts"}, concat(
			m(get("List posts", jp+"/posts")),
			rangeGet("Get post", jp+"/posts/%d", 1, 16),
			m(
				get("Post comments", jp+"/posts/1/comments"),
				post("Create post", jp+"/posts", map[string]any{"title": "hello", "body": "from senda", "userId": 1}),
				put("Replace post", jp+"/posts/1", map[string]any{"id": 1, "title": "replaced", "body": "x", "userId": 1}),
				patch("Patch post", jp+"/posts/1", map[string]any{"title": "patched"}),
				del("Delete post", jp+"/posts/1"),
			),
		)},
		{[]string{"JSONPlaceholder", "Comments"}, concat(
			m(get("List comments", jp+"/comments")),
			rangeGet("Get comment", jp+"/comments/%d", 1, 16),
			m(get("Comments by post", jp+"/comments?postId=2")),
		)},
		{[]string{"JSONPlaceholder", "Albums & Photos"}, concat(
			m(get("List albums", jp+"/albums")),
			rangeGet("Get album", jp+"/albums/%d", 1, 9),
			rangeGet("Album photos", jp+"/albums/%d/photos", 1, 9),
			m(get("List photos", jp+"/photos")),
			rangeGet("Get photo", jp+"/photos/%d", 1, 5),
		)},
		{[]string{"JSONPlaceholder", "Todos & Users"}, concat(
			m(get("List todos", jp+"/todos")),
			rangeGet("Get todo", jp+"/todos/%d", 1, 16),
			m(get("List users", jp+"/users")),
			rangeGet("Get user", jp+"/users/%d", 1, 11),
			m(
				get("User albums", jp+"/users/1/albums"),
				get("User todos", jp+"/users/1/todos"),
			),
		)},

		// ---- httpbin ----
		{[]string{"httpbin", "Methods"}, m(
			get("GET", hb+"/get"),
			post("POST", hb+"/post", map[string]any{"hello": "world"}),
			put("PUT", hb+"/put", map[string]any{"hello": "world"}),
			patch("PATCH", hb+"/patch", map[string]any{"hello": "world"}),
			del("DELETE", hb+"/delete"),
			get("Anything", hb+"/anything?q=senda"),
		)},
		{[]string{"httpbin", "Status codes"}, func() []model.Request {
			var out []model.Request
			for _, c := range []int{200, 201, 204, 301, 400, 401, 403, 404, 418, 500, 503} {
				out = append(out, get(fmt.Sprintf("Status %d", c), fmt.Sprintf("%s/status/%d", hb, c)))
			}
			return out
		}()},
		{[]string{"httpbin", "Request inspection"}, m(
			get("Headers", hb+"/headers"),
			get("IP", hb+"/ip"),
			get("User-Agent", hb+"/user-agent"),
			get("Query params", hb+"/get?foo=bar&baz=qux"),
		)},
		{[]string{"httpbin", "Response formats"}, m(
			get("JSON", hb+"/json"),
			get("XML", hb+"/xml"),
			get("HTML", hb+"/html"),
			get("UUID", hb+"/uuid"),
			get("Base64 decode", hb+"/base64/aGVsbG8gc2VuZGE="),
			get("Gzip", hb+"/gzip"),
			get("Deflate", hb+"/deflate"),
		)},
		{[]string{"httpbin", "Redirects & delay"}, m(
			get("Redirect 2", hb+"/redirect/2"),
			get("Relative redirect", hb+"/relative-redirect/3"),
			get("Delay 1s", hb+"/delay/1"),
			get("Delay 2s", hb+"/delay/2"),
		)},
		{[]string{"httpbin", "Auth & cookies"}, m(
			get("Basic auth", hb+"/basic-auth/user/passwd"),
			get("Bearer", hb+"/bearer"),
			get("Set cookie", hb+"/cookies/set?token=abc"),
			get("List cookies", hb+"/cookies"),
		)},

		// ---- Postman Echo ----
		{[]string{"Postman Echo"}, m(
			get("GET", pe+"/get?foo=bar"),
			post("POST json", pe+"/post", map[string]any{"name": "senda"}),
			get("Headers", pe+"/headers"),
			get("Response status 200", pe+"/status/200"),
			get("Delay 1s", pe+"/delay/1"),
			get("Time now", pe+"/time/now"),
			get("Basic auth", pe+"/basic-auth"),
			get("IP", pe+"/ip"),
		)},

		// ---- GitHub ----
		{[]string{"GitHub", "Users"}, m(
			get("Get octocat", gh+"/users/octocat"),
			get("Octocat repos", gh+"/users/octocat/repos"),
			get("Octocat followers", gh+"/users/octocat/followers"),
			get("Octocat gists", gh+"/users/octocat/gists"),
		)},
		{[]string{"GitHub", "Repos"}, m(
			get("Get golang/go", gh+"/repos/golang/go"),
			get("Go languages", gh+"/repos/golang/go/languages"),
			get("Go contributors", gh+"/repos/golang/go/contributors"),
			get("Go releases", gh+"/repos/golang/go/releases"),
			get("Linux topics", gh+"/repos/torvalds/linux/topics"),
		)},
		{[]string{"GitHub", "Meta"}, m(
			get("Rate limit", gh+"/rate_limit"),
			get("Zen", gh+"/zen"),
			get("Emojis", gh+"/emojis"),
			get("Search repos: go http", gh+"/search/repositories?q=http+language:go&per_page=5"),
		)},

		// ---- PokeAPI ----
		{[]string{"PokeAPI", "Pokemon"}, concat(
			m(get("List pokemon", pk+"/pokemon?limit=20")),
			func() []model.Request {
				var out []model.Request
				for _, n := range []string{"pikachu", "ditto", "charizard", "bulbasaur", "mewtwo", "eevee", "snorlax", "gengar"} {
					out = append(out, get("Get "+n, pk+"/pokemon/"+n))
				}
				return out
			}(),
			rangeGet("Get pokemon #", pk+"/pokemon/%d", 1, 16),
			m(get("Pikachu encounters", pk+"/pokemon/pikachu/encounters")),
		)},
		{[]string{"PokeAPI", "Metadata"}, m(
			get("Ability: static", pk+"/ability/static"),
			get("Type: electric", pk+"/type/electric"),
			get("Berry: cheri", pk+"/berry/cheri"),
			get("Species: pikachu", pk+"/pokemon-species/pikachu"),
			get("Move: thunderbolt", pk+"/move/thunderbolt"),
			get("Item: poke-ball", pk+"/item/poke-ball"),
			get("Region: kanto", pk+"/region/kanto"),
			get("Generation 1", pk+"/generation/1"),
		)},

		// ---- REST Countries ----
		{[]string{"REST Countries"}, m(
			get("All (fields)", rc+"/all?fields=name,capital,region"),
			get("By name: finland", rc+"/name/finland"),
			get("By code: FI", rc+"/alpha/FI"),
			get("By region: europe", rc+"/region/europe"),
			get("By currency: eur", rc+"/currency/eur"),
			get("By language: spanish", rc+"/lang/spanish"),
			get("By capital: tokyo", rc+"/capital/tokyo"),
		)},

		// ---- Open-Meteo ----
		{[]string{"Open-Meteo"}, m(
			get("Helsinki forecast", om+"/forecast?latitude=60.17&longitude=24.94&current=temperature_2m"),
			get("Tokyo forecast", om+"/forecast?latitude=35.68&longitude=139.69&hourly=temperature_2m"),
			get("London daily", om+"/forecast?latitude=51.5&longitude=-0.12&daily=temperature_2m_max&timezone=UTC"),
			get("Geocode: berlin", "https://geocoding-api.open-meteo.com/v1/search?name=Berlin&count=5"),
			get("Air quality", "https://air-quality-api.open-meteo.com/v1/air-quality?latitude=52.5&longitude=13.4&hourly=pm10"),
		)},

		// ---- CoinGecko ----
		{[]string{"CoinGecko"}, m(
			get("Ping", cg+"/ping"),
			get("Simple price BTC/ETH", cg+"/simple/price?ids=bitcoin,ethereum&vs_currencies=usd,eur"),
			get("Coins markets", cg+"/coins/markets?vs_currency=usd&per_page=10"),
			get("Bitcoin detail", cg+"/coins/bitcoin"),
			get("Bitcoin market chart", cg+"/coins/bitcoin/market_chart?vs_currency=usd&days=7"),
			get("Trending", cg+"/search/trending"),
			get("Global", cg+"/global"),
			get("Supported vs currencies", cg+"/simple/supported_vs_currencies"),
		)},

		// ---- Frankfurter ----
		{[]string{"Frankfurter FX"}, m(
			get("Latest", fr+"/latest"),
			get("Latest EUR->USD,GBP", fr+"/latest?from=EUR&to=USD,GBP"),
			get("Historical 2020-01-01", fr+"/2020-01-01"),
			get("Time series", fr+"/2024-01-01..2024-01-31?to=USD"),
			get("Currencies", fr+"/currencies"),
		)},

		// ---- Rick and Morty ----
		{[]string{"Rick and Morty"}, concat(
			m(get("Characters", rm+"/character")),
			rangeGet("Character", rm+"/character/%d", 1, 16),
			m(get("Locations", rm+"/location")),
			rangeGet("Location", rm+"/location/%d", 1, 6),
			m(get("Episodes", rm+"/episode")),
			rangeGet("Episode", rm+"/episode/%d", 1, 11),
			m(get("Filter: alive humans", rm+"/character/?status=alive&species=human")),
		)},

		// ---- Open Library ----
		{[]string{"Open Library"}, m(
			get("Search: tolkien", ol+"/search.json?q=tolkien&limit=5"),
			get("Search by title", ol+"/search.json?title=the+hobbit"),
			get("Work: OL45883W", ol+"/works/OL45883W.json"),
			get("Author: OL26320A", ol+"/authors/OL26320A.json"),
			get("ISBN 9780140328721", ol+"/isbn/9780140328721.json"),
			get("Subject: love", ol+"/subjects/love.json?limit=5"),
		)},

		// ---- ReqRes ----
		{[]string{"ReqRes"}, m(
			get("List users p1", rr+"/users?page=1"),
			get("List users p2", rr+"/users?page=2"),
			get("Single user 2", rr+"/users/2"),
			get("User not found", rr+"/users/23"),
			get("List resources", rr+"/unknown"),
			post("Create user", rr+"/users", map[string]any{"name": "senda", "job": "client"}),
			put("Update user", rr+"/users/2", map[string]any{"name": "senda", "job": "tester"}),
			post("Register", rr+"/register", map[string]any{"email": "eve.holt@reqres.in", "password": "pistol"}),
			post("Login", rr+"/login", map[string]any{"email": "eve.holt@reqres.in", "password": "cityslicka"}),
			get("Delayed response", rr+"/users?delay=3"),
		)},

		// ---- FakeStore ----
		{[]string{"FakeStore", "Products"}, m(
			get("All products", fs+"/products"),
			get("Product 1", fs+"/products/1"),
			get("Limit 5", fs+"/products?limit=5"),
			get("Categories", fs+"/products/categories"),
			get("In electronics", fs+"/products/category/electronics"),
			post("Add product", fs+"/products", map[string]any{"title": "thing", "price": 9.99, "category": "misc"}),
		)},
		{[]string{"FakeStore", "Carts & Users"}, m(
			get("All carts", fs+"/carts"),
			get("Cart 1", fs+"/carts/1"),
			get("User carts", fs+"/carts/user/2"),
			get("All users", fs+"/users"),
			get("User 1", fs+"/users/1"),
			post("Login", fs+"/auth/login", map[string]any{"username": "mor_2314", "password": "83r5^_"}),
		)},

		// ---- Fun & Facts ----
		{[]string{"Fun & Facts", "Animals"}, m(
			get("Cat fact", "https://catfact.ninja/fact"),
			get("Cat facts list", "https://catfact.ninja/facts?limit=5"),
			get("Random dog", "https://dog.ceo/api/breeds/image/random"),
			get("Dog breeds", "https://dog.ceo/api/breeds/list/all"),
			get("Husky images", "https://dog.ceo/api/breed/husky/images"),
			get("Random fox", "https://randomfox.ca/floof/"),
		)},
		{[]string{"Fun & Facts", "Jokes & Quotes"}, m(
			get("Chuck Norris", "https://api.chucknorris.io/jokes/random"),
			get("Chuck categories", "https://api.chucknorris.io/jokes/categories"),
			get("Official joke", "https://official-joke-api.appspot.com/random_joke"),
			get("Kanye quote", "https://api.kanye.rest/"),
			get("Advice slip", "https://api.adviceslip.com/advice"),
			get("Useless fact", "https://uselessfacts.jsph.pl/api/v2/facts/random"),
			get("Bored activity", "https://www.boredapi.com/api/activity"),
		)},
		{[]string{"Fun & Facts", "Names & Numbers"}, m(
			get("Agify: michael", "https://api.agify.io/?name=michael"),
			get("Genderize: anna", "https://api.genderize.io/?name=anna"),
			get("Nationalize: sven", "https://api.nationalize.io/?name=sven"),
			get("Random user", "https://randomuser.me/api/"),
			get("3 random users", "https://randomuser.me/api/?results=3"),
			get("Number trivia 42", "http://numbersapi.com/42?json"),
			get("Math fact 7", "http://numbersapi.com/7/math?json"),
			get("Date fact", "http://numbersapi.com/2/29/date?json"),
		)},

		// ---- Time & Holidays ----
		{[]string{"Time & Holidays"}, m(
			get("World time UTC", "https://worldtimeapi.org/api/timezone/Etc/UTC"),
			get("World time Helsinki", "https://worldtimeapi.org/api/timezone/Europe/Helsinki"),
			get("Timezone list", "https://worldtimeapi.org/api/timezone"),
			get("Current UTC (timeapi)", "https://timeapi.io/api/Time/current/zone?timeZone=UTC"),
			get("FI holidays 2025", "https://date.nager.at/api/v3/PublicHolidays/2025/FI"),
			get("US holidays 2025", "https://date.nager.at/api/v3/PublicHolidays/2025/US"),
			get("Available countries", "https://date.nager.at/api/v3/AvailableCountries"),
			get("Next holidays worldwide", "https://date.nager.at/api/v3/NextPublicHolidaysWorldwide"),
		)},

		// ---- Space ----
		{[]string{"Space"}, m(
			get("ISS position", "http://api.open-notify.org/iss-now.json"),
			get("People in space", "http://api.open-notify.org/astros.json"),
			get("SpaceX latest launch", "https://api.spacexdata.com/v5/launches/latest"),
			get("SpaceX rockets", "https://api.spacexdata.com/v4/rockets"),
			get("SpaceX next launch", "https://api.spacexdata.com/v5/launches/next"),
		)},

		// ---- Deck of Cards ----
		{[]string{"Deck of Cards"}, m(
			get("New shuffled deck", dc+"/new/shuffle/?deck_count=1"),
			get("New deck", dc+"/new/"),
			get("Draw 5 (new)", dc+"/new/draw/?count=5"),
		)},

		// ---- GraphQL ----
		{[]string{"GraphQL"}, m(
			gql("Countries: by code", "https://countries.trevorblades.com/",
				"query($code: ID!){ country(code:$code){ name capital currency emoji } }", "{\n  \"code\": \"FI\"\n}"),
			gql("Countries: continents", "https://countries.trevorblades.com/",
				"{ continents { code name } }", ""),
			gql("Rick&Morty: characters", "https://rickandmortyapi.com/graphql",
				"{ characters(page:1){ results { name status species } } }", ""),
			gql("SpaceX: launches", "https://spacex-production.up.railway.app/",
				"{ launchesPast(limit:3){ mission_name launch_year } }", ""),
		)},
	}
}

// ---- history seeding -------------------------------------------------------

type bucket struct {
	age    time.Duration
	status int
	errMsg string
}

func buckets() []bucket {
	return []bucket{
		{3 * time.Minute, 200, ""},
		{25 * time.Minute, 200, ""},
		{3 * time.Hour, 200, ""},
		{9 * time.Hour, 201, ""},
		{2 * 24 * time.Hour, 200, ""},
		{6 * 24 * time.Hour, 304, ""},
		{20 * 24 * time.Hour, 200, ""},
		{1 * time.Hour, 500, ""},
		{40 * time.Minute, 0, "dial tcp: connection refused"},
	}
}

func main() {
	dest := "examples-collection/large"
	if len(os.Args) > 1 {
		dest = os.Args[1]
	}

	if err := os.RemoveAll(dest); err != nil {
		fail(err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		fail(err)
	}
	// Establish the .senda/ config dir so root metadata, environments and
	// history are written into it rather than at the collection root.
	store.Migrate(dest)

	// Root metadata + environments.
	if err := store.SaveCollection(model.Collection{Path: dest, Name: "large", Auth: model.Auth{Type: model.AuthNone}}); err != nil {
		fail(err)
	}
	if err := store.SaveEnvironment(dest, model.Environment{Name: "dev", Vars: []model.KV{{Key: "BASE", Value: "https://httpbin.org", Enabled: true}}}); err != nil {
		fail(err)
	}
	if err := store.SaveEnvironment(dest, model.Environment{Name: "prod", Vars: []model.KV{{Key: "BASE", Value: "https://api.github.com", Enabled: true}}}); err != nil {
		fail(err)
	}

	// Write every request, recording (method,url) for history seeding.
	type ref struct{ method, url string }
	var refs []ref
	count := 0
	folders := spec()
	for _, f := range folders {
		dir := dest
		for _, s := range f.segs {
			dir = filepath.Join(dir, safe(s))
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fail(err)
		}
		for _, r := range f.reqs {
			path := filepath.Join(dir, safe(r.Name)+".yaml")
			if err := store.SaveRequest(path, r); err != nil {
				fail(err)
			}
			refs = append(refs, ref{r.Method, r.URL})
			count++
		}
	}

	// Seed history for ~1 in 4 requests, cycling through recency buckets, and
	// append oldest-first so the log reads newest-last on disk (matching how
	// the app writes it). The reference time is fixed (not time.Now) so the
	// generated collection — and thus the committed .zip — is reproducible.
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bs := buckets()
	type seeded struct {
		at    time.Time
		entry model.HistoryEntry
	}
	var seeds []seeded
	const step = 4
	for i, rf := range refs {
		if i%step != 0 {
			continue
		}
		b := bs[(i/step)%len(bs)]
		at := now.Add(-b.age)
		seeds = append(seeds, seeded{at, model.HistoryEntry{
			At:         at.Format(time.RFC3339),
			Method:     rf.method,
			URL:        rf.url,
			Status:     b.status,
			DurationMs: int64(30 + i%200),
			SizeBytes:  int64(120 + (i*7)%4000),
			Error:      b.errMsg,
		}})
	}
	sort.Slice(seeds, func(i, j int) bool { return seeds[i].at.Before(seeds[j].at) })
	for _, s := range seeds {
		if err := history.Append(dest, s.entry); err != nil {
			fail(err)
		}
	}

	fmt.Printf("Wrote %d requests across %d folders to %s\n", count, len(folders), dest)
	fmt.Printf("Seeded %d history entries (.senda/history.jsonl)\n", len(seeds))

	// Pack into a reproducible .senda.zip committed to git: a single binary blob
	// keeps diffs tiny vs hundreds of YAML files. The working tree stays
	// gitignored.
	zipPath := strings.TrimRight(dest, "/") + store.ArchiveExt
	if err := store.PackDir(dest, zipPath); err != nil {
		fail(err)
	}
	fmt.Printf("Packed collection to %s\n", zipPath)
}

// safe makes a string usable as a path segment (Senda allows spaces).
func safe(s string) string { return strings.ReplaceAll(s, "/", "-") }

func fail(err error) {
	fmt.Fprintln(os.Stderr, "genexamples:", err)
	os.Exit(1)
}
