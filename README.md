# ghLabeler
This webhook utility that will __automatically update Github issue labels__ when an issue/PR card is moved from one column of your __project board__ to the next. 

## Deploying
When deploying utility you need to provide several variables:
- `ACCESS_TOKEN` GitHub's personal access token
- `WEBHOOK_TOKEN` Token used for request signing
- `PROJECT_NAMES` comma separated list of project names, that will be affected by this utility. __Note__ Project must be in repo. It does not work with ORG level projects. eg `Sample Project,Another project`
- `PORT` port on which utility will listen

when deployed utility will listen ar root url: `http(s)://somehost.com/`
If you don't have a web server to deploy the script on, use a free instance of [Heroku](https://www.heroku.com/).

## Configuring
When you have deployed `ghLabeler` you need to set up your repo to work with it.
1. Create project(s) named as entries in `PROJECT_NAMES` variable on your server.
1. Setup columns.
    - Add to name of your columns `lables:` tag with comma separated list of lable names to use when card gets into that column. eg `lables:New,Web,Test`
    - Add to name of your columns `users:` tag with comma separated list of user names to assign to the card when it gets into that column. eg `Users:DrMagPie,Conor,Rick`
    - __Note__ these list are not required. You can use only one of them.
1. setup webhook
    - _Content type_ must be set to `application/json`
    - As the _trigger_ select the _event_ `Project card`.


Now everything is ready to be used. When you move an issue card in your Github Project from one column to another, the labels from the column names `lables:` list will be added and removed. And cards will be assigned and unassigned to the persons in `users:`  list.

## Feedback

If you find any issues in the Utility, feel free to open an issue or send me a pull request.

