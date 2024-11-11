`kaizer` is a tool for walking and talking, for thinking on the move.

## motivation

I like to think out loud. I prefer to walk and talk rather than sit at a desk and write or type. Each mode has its advantages but, for a very long time, text-at-a-desk has been the practical choice for most serious thinking. I think the situation is changing rapidly, and I'm super excited.

## contents

- [motivation](#motivation)
- [contents](#contents)
- [kaizer?](#kaizer)
- [features](#features)
- [down the road](#down-the-road)

## kaizer?

Kaizer is my son's name. I like it, it's easy to say, and easy to remember.

## features

Right now `kaizer` is a single-user, single-file web server which integrates with the Twilio API in order to stream inbound calls' audio into an S3 bucket.

## down the road

Development will progress according to a pair of high-level goals and considerations:

1. From a user's perspective, I personally need an MVP, a frictionless way to stream audio to a programmable backend, immediately.
2. From a developer/hobbyist perspective, I'm personally enamored by the idea of very simple online services made profitable through high-quality engineering.
3. From a technologist's perspective, audio interfaces paired with language models are going to completely reshape the world, and I want to participate. It sounds silly to say, but I believe this is still an underappreciated idea. How much of how we all live day in and day out is shaped by the constraint of having to sit behind a desk?

For the most part (1) is satisfied. To satisfy (2) I'll need to implement basic multi-user functionality, integrate with Stripe, and design and implement a minimal feature set and frontend. The feature set is going to be something like

1. A download/export API
2. A configureable webhooks + streaming API
3. A configureable multi-backend-phone-number system
