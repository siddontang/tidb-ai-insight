Use your nature language to get the insight in your database

- Register TiDB cloud Serverless Tier: https://tidb.cloud/
- Register Open AI and create an API Key: https://beta.openai.com/account/api-keys

## Run

Enjoy getting insight in default gharchive_dev database in TiDB cloud Serverless Tier. 


```bash
go run main.go -H {host} -P {port} -u {user} -p {password} --key {OpenAI API toekn} 
prompt> who contributed most prs
CREATE TABLE temp_author_prs AS
SELECT actor_id, actor_login, COUNT(pr_or_issue_id) AS pr_contributions
FROM github_events
WHERE type = 'PullRequestEvent'
GROUP BY actor_id, actor_login
ORDER BY pr_contributions DESC
prompt>
```