# lambda-ses-forwarder
AWS SES Email forwarder Lambda function written in Go.

This is a Lambda function to forward emails received from SES.

## Setup instructions

Assuming you have SES incoming emails enabled, this function needs the following configuration.

### IAM

* Create a Lambda execution role with a policy that allows
    * _s3:GetObject_ on a bucket you can store incoming e-mails in
    * _ses:SendRawEmail_ to forward the email to the new destination
    * Don't forget to include the default Lambda execution policy to be able to write Cloudwatch logs

### Lambda

* Create a Lambda function with Go 1.x runtime and the execution role you have just created
* Build this program or download a [release](https://github.com/SebastiaanKlippert/lambda-ses-forwarder/releases) zip and upload it as source
* Set the exection handler name to the name of the binary (lambda_ses_forwarder_linux if you use a release)
* Memory size can be as low as 128MB if you don't receive large mails, but 256MB or higher is recommended
* You can configure the function by using these environment variables:

Variable name | Required | Description
--- | --- | ---
FORWARD_FROM | Yes | Address<sup>1</sup> used as FROM address when forwarding mail<sup>2</sup>
FORWARD_TO | Yes | Address<sup>1</sup> used as TO address when forwarding mail
S3_BUCKET | Yes | Name of the S3 bucket you use to store incoming mails
S3_BUCKET_REGION | No | Can be set when the S3 bucket is in a different AWS region than your Lambda
S3_PREFIX | No | If you use a key prefix when storing mails you can enter it here

<sup>1</sup> a single RFC 5322 address, e.g. `test@example.com` or `My Name <test@example.com>` 

<sup>2</sup> can include %s as replacer to include the original sender name, e.g. `%s through my mail forwarder <forwarder@example.com>`

**Important** Ensure the FROM address is a verified address in SES and is allowed to send mail, the original FROM address will be set as Reply-To address so you can answer the mails directly.

![Lambda Env](https://sklippert.s3-eu-central-1.amazonaws.com/public/lambda-env.png "Lambda environment")


### SES

* Create a Rule Set for the incoming address or domain you want to forward emails for
* Create a first action to store your mail in the S3 bucket from the previous steps with an optional prefix
* Create a second action to call the Lambda function you created

![SES Rule](https://sklippert.s3-eu-central-1.amazonaws.com/public/ses-rule.png "SES Rule")



