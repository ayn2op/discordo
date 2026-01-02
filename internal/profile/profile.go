package profile

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/diamondburned/arikawa/v3/discord"
)

type UserProfile struct {
	Pronouns string `json:"pronouns"`
}

type profileResponse struct {
	UserProfile UserProfile `json:"user_profile"`
}

type Cache struct {
	client *http.Client
	token  string

	mu       sync.RWMutex
	profiles map[discord.UserID]string
}

func NewCache(client *http.Client, token string) *Cache {
	return &Cache{
		client:   client,
		token:    token,
		profiles: make(map[discord.UserID]string),
	}
}

func (c *Cache) Get(userID discord.UserID) (string, bool) {
	c.mu.RLock()
	pronouns, ok := c.profiles[userID]
	c.mu.RUnlock()
	return pronouns, ok
}

func (c *Cache) Fetch(userID discord.UserID) string {
	c.mu.RLock()
	if pronouns, ok := c.profiles[userID]; ok {
		c.mu.RUnlock()
		return pronouns
	}
	c.mu.RUnlock()

	pronouns := c.fetchFromAPI(userID)

	c.mu.Lock()
	c.profiles[userID] = pronouns
	c.mu.Unlock()

	return pronouns
}

func (c *Cache) fetchFromAPI(userID discord.UserID) string {
	url := fmt.Sprintf("https://discord.com/api/v9/users/%s/profile?with_mutual_guilds=false&with_mutual_friends=false", userID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("Authorization", c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var profile profileResponse
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return ""
	}

	return profile.UserProfile.Pronouns
}

func (c *Cache) FetchMany(userIDs []discord.UserID, onComplete func()) {
	var toFetch []discord.UserID

	c.mu.RLock()
	for _, id := range userIDs {
		if _, ok := c.profiles[id]; !ok {
			toFetch = append(toFetch, id)
		}
	}
	c.mu.RUnlock()

	if len(toFetch) == 0 {
		return
	}

	go func() {
		for _, id := range toFetch {
			c.Fetch(id)
		}
		if onComplete != nil {
			onComplete()
		}
	}()
}
