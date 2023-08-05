package poe

import (
	"errors"
	"sync"
	"time"
	"log"
	"net/url"
	"github.com/juzeon/poe-openai-proxy/conf"
 
	"github.com/juzeon/poe-openai-proxy/poeapi"
	"github.com/juzeon/poe-openai-proxy/util"
	// poeapi "github.com/lwydyby/poe-api"
)

var clients []*Client
var clientLock sync.Mutex
var clientIx = 0

var tokenMutex sync.Mutex
var correctTokens []string
var errorTokens []string

func createClient(token string, wg *sync.WaitGroup) {
	defer func() {
		if r := recover(); r != nil {
			util.Logger.Error("Recovered in NewClient: %v\n", r)
			tokenMutex.Lock()
			defer tokenMutex.Unlock()
			errorTokens = append(errorTokens, token)
		}
	}()
	defer wg.Done()

	client, err := NewClient(token, conf.Conf.Proxy)
	if err != nil || client == nil  {
		util.Logger.Error("Error creating client with token %s: %v", token, err)
		tokenMutex.Lock()
		defer tokenMutex.Unlock()
		errorTokens = append(errorTokens, token)
		return
	}

	tokenMutex.Lock()
	correctTokens = append(correctTokens, token)
	clients = append(clients, client)
	tokenMutex.Unlock()
}

func Setup() {

 
	log.Printf("load proxy is %s", conf.Conf.Proxy )

	seen := make(map[string]bool)
	wg := sync.WaitGroup{}
	wg.Add(len(conf.Conf.Tokens))

	for _, token := range conf.Conf.Tokens {
		if seen[token] {
			wg.Done()
			continue
		}
		seen[token] = true

		go createClient(token, &wg)
		 
		time.Sleep(1) 
	}
	wg.Wait()

	// Log the correct and error tokens as lists
	util.Logger.Info("Success tokens:", correctTokens)
	util.Logger.Error("Error tokens:", errorTokens)

}

type Client struct {
	Token  string
	client *poeapi.Client
	Usage  []time.Time
	Lock   bool
}

func NewClient(token string, proxy string ) (*Client, error) {
	log.Printf("registering client: %v, proxy %v " ,  token ,  proxy )
	var proxyUrl * url.URL = nil 
	if len(proxy ) >  0   {		
		proxyUrl,_ = url.Parse(proxy)
	}
	
	log.Printf("using proxy %v", proxy )
	client := poeapi.NewClient(token, proxyUrl  )
	return &Client{Token: token, Usage: nil, Lock: false, client: client}, nil
}


func (c *Client) getContentToSend(messages []Message) string {
	leadingMap := map[string]string{
		"system":    "Instructions",
		"user":      "User",
		"assistant": "Assistant",
	}
	content := ""
	var simulateRoles bool
	switch conf.Conf.SimulateRoles {
	case 0:
		simulateRoles = false
	case 1:
		simulateRoles = true
	case 2:
		if len(messages) == 1 && messages[0].Role == "user" ||
			len(messages) == 1 && messages[0].Role == "system" ||
			len(messages) == 2 && messages[0].Role == "system" && messages[1].Role == "user" {
			simulateRoles = false
		} else {
			simulateRoles = true
		}
	}
	for _, message := range messages {
		if simulateRoles {
			content += "||>" + leadingMap[message.Role] + "-\n" + message.Content + "\n"
		} else {
			content += message.Content + "\n"
		}
	}
	if simulateRoles {
		content += "||>Assistant-\n"
	}
	util.Logger.Debug("Generated content to send: " + content)
	return content
}

func (c *Client) Stream(messages []Message, model string) (<-chan string, error) {
	channel := make(chan string, 1024)
	content := c.getContentToSend(messages)

	util.Logger.Info("using model ", model )
	bot, ok := conf.Conf.Bot[model]
	if !ok {
		bot = "capybara"
	}
	util.Logger.Info("Stream using bot", bot)
	if c.client == nil {
		util.Logger.Error("invalid client", bot )
		return nil , errors.New("invalid client")
	}
	resp, err := c.client.SendMessage(bot, content, true, time.Duration(conf.Conf.Timeout)*time.Second)
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(channel)
		defer func() {
			if err := recover(); err != nil {
				channel <- "\n\n[ERROR] " + err.(error).Error()
			}
		}()
		for message := range poeapi.GetTextStream(resp) {
			channel <- message
		}
		channel <- "[DONE]"
	}()
	return channel, nil
}

func (c *Client) Ask(messages []Message, model string) (*Message, error) {
	content := c.getContentToSend(messages)

	bot, ok := conf.Conf.Bot[model]
	if !ok {
		bot = "capybara"
	}
	util.Logger.Info("Ask using bot", bot)
	if c == nil {
		return nil , errors.New("nil client ")
	}

	resp, err := c.client.SendMessage(bot, content, true, time.Duration(conf.Conf.Timeout)*time.Second)
	if err != nil {

		return nil, err
	}
	return &Message{
		Role:    "assistant",
		Content: poeapi.GetFinalResponse(resp),
		Name:    "",
	}, nil
}

func (c *Client) Release() {
	clientLock.Lock()
	defer clientLock.Unlock()
	c.Lock = false
}

func GetClient() (*Client, error) {
	clientLock.Lock()
	defer clientLock.Unlock()
	if len(clients) == 0 {
		return nil, errors.New("no client is available")
	}
	for i := 0; i < len(clients); i++ {
		client := clients[clientIx%len(clients)]
		clientIx++
		if client.Lock {
			continue
		}
		if len(client.Usage) > 0 {
			lastUsage := client.Usage[len(client.Usage)-1]
			if time.Since(lastUsage) < time.Duration(conf.Conf.CoolDown)*time.Second {
				continue
			}
		}
		if len(client.Usage) < conf.Conf.RateLimit {
			client.Usage = append(client.Usage, time.Now())
			client.Lock = true
			return client, nil
		} else {
			usage := client.Usage[len(client.Usage)-conf.Conf.RateLimit]
			if time.Since(usage) <= 1*time.Minute {
				continue
			}
			client.Usage = append(client.Usage, time.Now())
			client.Lock = true
			return client, nil
		}
	}
	return nil, errors.New("no available client")
}
