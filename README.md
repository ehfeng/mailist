# Mailist

> Personable mailing lists

```html
<form action="mailist.me/user/list?next=redirect-url" method="POST">
  <input name="email" type="email" />
  <button>Subscribe</button>
</form>
```

This doesn't send the emails, but lets you copy the emails to the To field. We warn you if the list grows to over 500 subscribers.

The worse case is you send a few people the same thing twice.

CORS, Re-captcha's, de-duplicates, validates with NeverBounce and Mailgun confirmation emails

You can add emails manually and those do not need to be validated.

List can be copied in. Controls (unsub)


```
POST mailist.me/<user>/<list>

email: me@ehfeng.com
```

```
GET mailist.me/<user>/<list>/unsubscribe?email=<email>&token=<token>
```

Token is a hash of the email, username, listname, and a salt.

Lambda?
