# awesome-go-sync

To make sure [awesome-go-bot](github.com/samirkape/awesome-go-bot) always have the updated information such as new packages and star counts, 

this daemon service runs itself once every day.


### Stack
* Go 1.16
* MongoDB
* Github API
* AWS Lambda
* AWS CloudWatch (for daily trigger)
