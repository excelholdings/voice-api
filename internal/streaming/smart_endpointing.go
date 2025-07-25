package streaming

import (
	"context"
	"encoding/json"
	"github.com/flyflow-devs/flyflow/internal/logger"
	"github.com/sashabaranov/go-openai"
	"time"
)

type ProbabilityResponse struct {
	Probability uint `json:"probability"`
}

func (c *CallOrchestrator) smartEndpointing(messages []openai.ChatCompletionMessage, probabilityChan chan uint) {
	prompt := `You are being used in an interactive voice application that sometimes returns partial responses. Your job is to return the probability that the user has completed a full thought and now wants the agent to respond

The probability you assign will be used to calculate how long the agent waits before confirming the user has finished their thought and the agent can respond.

INSTRUCTIONS

- You will receive a response from a user, it might come in several user messages or just one
- If the user responds with a full sentence to a non open ended question, assign a high probability
- You will classify the probability that the user is done talking, based on the last few messages in the conversation
- Sometimes you'll get partial responses, if the response seems like it's cut off or not finished assign it a lower probability
- If the assistant asks for a list, make sure they list multiple items before finalizing your response
- If the assistant asks an open ended question and the user responds with a full sentence, assign a lower probability to give them more time to finish their thought
- When the assistant asks an open ended question, assign the probability in the middle (30-60) so that the user has more time to respond (even if the user has responded with multiple sentences)
- You will return a JSON blob {"probability": <result uint>}, where the result is an integer between 0 and 100 

HIGH PROBABILITY USER HAS FINISHED EXAMPLES

Assistant: Hey, this is Gary with TaxSlayer off-hours support. How can I help you?
User: Hey, Gary. I'm trying to
User: fill out, this
User: box four zero two a.
{"probability": 90}

Assistant: Got it. Is this related to a Roth Distribution held for over 5 years, nondeductible Traditional IRA contributions, a direct rollover between qualified plans, or are you a Public Safety Officer? Or none of these?
User: I'm a public safety officer. Does that mean anything?
{"probability": 91}

Assistant: How can I assist you further?
User: Hello?
{"probability": 91}

Assistant: Responses are typically generated in less than a second.
User: Hello?
{"probability": 91}

Assistant: How can I help you today?
User: I'm just testing again
{"probability": 95}

Assistant: Yes, it does. You'll need to complete a separate worksheet. Would you like me to email you the link?
User: Yes, please.
{"probability": 99}

Assistant: Great! Please provide me with your email address
User: It's preppy at fly flow dot dev
{"probability": 95}

Assistant: Thank you! I'll send over the link to that worksheet right away. Anything else I can assist you with?
User: Nope. That's it. Thanks, Gary.
{"probability": 89}

Assistant: Hey there, tell me about whats going on with your company today?
User: Hello?
{"probability": 80}

PARTIAL RESPONSES EXAMPLES 
Assistant: What's your opinion on the genevia convention
User: I think
{"probability": 9}

Assistant: I'd be happy to keep the conversation going. So, how's your day been so far? Have you made any progress on your startup or project? I'm here to listen and help if I can.
User: Yeah. I just
{"probability": 13}

Assistant: I'd be happy to keep the conversation going. So, how's your day been so far? Have you made any progress on your startup or project? I'm here to listen and help if I can.
User: Yeah. I just
User: implemented a new latency optimization that should make you faster.
{"probability": 70}

Assistant: What's your favorite dog?
User: I like corgis
{"probability": 92}

Assistant: how was your day today?
User: Um, I need to think about that some more
{"probability": 70}

Assistant: What are you building?
User: application for users to create agents, and then also making that
{"probability": 31}

Assistant: What are you building?
User: application for users to create agents, and then also making that
User: for a specific market segment, I guess. 
{"probability": 85}

Assistant: What route are you going?
User: Well, we could Gavin goes, like, a traditional route
{"probability": 52}

Assistant: What route are you going?
User: Well, we could Gavin goes, like, a traditional route
User: So something like general contractors, plumbers, like that
{"probability": 52}

Assistant: What do your users need?
User: Well, a lot of them have the need for
{"probability": 22}

Assistant: What do your users need?
User: Well, a lot of them have the need for
User: an AI voice mail feature
{"probability": 91}

Assistant: Got it, thanks for letting me know. How are you involved in your community or church?
User: I, volunteer with Habitat
{"probability": 22}

Assistant: Got it, thanks for letting me know. How are you involved in your community or church?
User: I, volunteer with Habitat
User: for Humanity and build houses
{"probability": 22}

Assistant: That's wonderful! Your involvement with Habitat for Humanity shows a strong commitment to helping others. Why do you think you would be a good fit for chick-fil-A, and what about the company values resonates with you?
User: I think I really like
{"probability": 22}

Assistant: That's wonderful! Your involvement with Habitat for Humanity shows a strong commitment to helping others. Why do you think you would be a good fit for chick-fil-A, and what about the company values resonates with you?
User: I think I really like
User: fil A because they focus on food quality, and that matters to me.
{"probability": 90}

EXAMPLE OF PARTIAL LISTS 

Assistant: List your top 5 favorite foods
User: Well I like bananas
{"probability": 15}

Assistant: List your top 5 favorite foods
User: Well I like bananas
User: ice cream, steak, apples, and fruit
{"probability": 91}

LONGER SENTENCES / OPEN ENDED QUESTIONS EXAMPLES
Assistant: What's your main objective for today's launch on Bookface?
User: Well, I want all the other YC founders to think we're really cool.
{"probability": 43}

Assistant: What key points does your demo highlight about your product's uniqueness and capabilities?
User: Well, hopefully, this call is natural sounding enough to impress
{"probability": 35}

Assistant: What key points does your demo highlight about your product's uniqueness and capabilities?
User: I'm showcasing how much time we've spent optimizing the
User: technology and all the traction that we have.
{"probability": 35}

Assistant: What makes each location special for you?
User: Well, I really like copper for it's back bowls and adventurous terrain.
User: A-Basin is good because it has a down to earth vibe.
User: Aspen has a lot of good restuarants I enjoy.
{"probability": 35}

Assistant: Thank you for confirming that. You mentioned on your application that you're free on Mondays and Tuesdays from 2 pm to 7 pm. Can you confirm if that still works for you?
User: Yeah. I'm still free on Mondays and Tuesdays, but I'm also
{"probability": 20}

Assistant: Thank you for confirming that. You mentioned on your application that you're free on Mondays and Tuesdays from 2 pm to 7 pm. Can you confirm if that still works for you?
User: Yeah. I'm still free on Mondays and Tuesdays, but I'm also
User: I'm in high school still.
{"probability": 80}
`

	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	defer cancel()

	openaiConfig := openai.DefaultConfig(c.cfg.FireworksAPIKey)
	openaiConfig.BaseURL = "https://api.fireworks.ai/inference/v1"
	openaiClient := openai.NewClientWithConfig(openaiConfig)

	fullMessages := []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: prompt,
		},
	}
	fullMessages = append(fullMessages, messages...)

	start := time.Now()

	respChan := make(chan openai.ChatCompletionResponse)
	errChan := make(chan error)

	go func() {
		resp, err := openaiClient.CreateChatCompletion(
			ctx,
			openai.ChatCompletionRequest{
				Model:    "accounts/fireworks/models/llama-v3-70b-instruct",
				Messages: fullMessages,
				ResponseFormat: &openai.ChatCompletionResponseFormat{
					Type: openai.ChatCompletionResponseFormatTypeJSONObject,
				},
			},
		)
		if err != nil {
			errChan <- err
			return
		}
		respChan <- resp
	}()

	select {
	case <-ctx.Done():
		logger.S.Warn("Request timed out after 700ms")
		probabilityChan <- 100
	case err := <-errChan:
		logger.S.Error(err)
		probabilityChan <- 100
	case resp := <-respChan:
		logger.S.Infof("time to generating endpointing %v", time.Since(start))

		var probabilityResp ProbabilityResponse
		err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &probabilityResp)
		if err != nil {
			probabilityChan <- 100
		} else {
			probabilityChan <- probabilityResp.Probability
		}
	}
}