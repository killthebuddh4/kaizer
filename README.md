`kaizer` is a tool for walking and talking, for thinking on the move.

## motivation

I like to think out loud. I prefer to walk and talk rather than sit at a desk and write or type. Each mode has its advantages but, for a very long time, text-at-a-desk has been the practical choice for most serious thinking. I think the situation is changing rapidly, and I'm super excited.

## contents

- [motivation](#motivation)
- [contents](#contents)
- [kaizer?](#kaizer)
- [Roadmap](#roadmap)

## kaizer?

Kaizer is my son's name. I like it, it's easy to say, and easy to remember.

## Roadmap

The very first step is to build a zero-friction tool for streaming audio to a programmable backend. At the moment, the minimum-friction way to record audio files on an iPhone is by wiring the action button to the voice memo app. The obvious downside is that there's no way (that I know of) to programmatically access the recordings. I haven't found an app nor a hardware device that basically just records audio and streams it to a user-defined backend. 

My current plan is to write a go server that uses the Twilio API/SDK to basically pipe received calls' audio into S3.

Update: I found an app called [Larix Broadcaster](https://softvelum.com/larix/) that looks like it might do some of what I want. I haven't tested it yet but I may be able to set up a WebRTC stream from my phone's microphone to my own WebRTC server. If I can wire the app to the action button, it might be good enough for now.

