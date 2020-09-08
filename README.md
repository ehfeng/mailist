# Mailist

> Personable mailing lists

Meant to be deployed on your own domain.

```html
<form action="example.com/listname?next=redirect-url" method="POST">
  <input name="email" />
  <button>Subscribe</button>
</form>
```

This doesn't send the emails, but lets you copy the emails to the To field. Gmail only allows for 500 emails per message AND per day.

You can add emails manually and those do not need to be validated.

# API

```
POST /list?next&referrer
Content-Type: application/x-www-form-urlencoded
```
