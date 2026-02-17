package email

import (
	"bytes"
	"html/template"
)

// TemplateData holds the common data passed to all email templates.
type TemplateData struct {
	ShopName       string
	CustomerName   string
	OrderNumber    string
	TrackingNumber string
	TrackingURL    string
	RefundAmount   string
	CartURL        string
	InviteURL      string
	InviterName    string
	Role           string
	Body           string
}

const baseLayout = `<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"></head>
<body style="margin:0;padding:0;font-family:Arial,Helvetica,sans-serif;background:#f4f4f7;color:#333;">
<table width="100%" cellpadding="0" cellspacing="0" style="background:#f4f4f7;padding:24px 0;">
<tr><td align="center">
<table width="580" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;overflow:hidden;">
<tr><td style="background:#2d3748;padding:20px 32px;">
<h1 style="margin:0;color:#ffffff;font-size:20px;">{{.ShopName}}</h1>
</td></tr>
<tr><td style="padding:32px;">
{{.Content}}
</td></tr>
<tr><td style="padding:16px 32px;background:#f7fafc;text-align:center;font-size:12px;color:#a0aec0;">
&copy; {{.ShopName}}. All rights reserved.
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>`

type layoutData struct {
	ShopName string
	Content  template.HTML
}

func renderTemplate(shopName, innerHTML string) (string, error) {
	tmpl, err := template.New("base").Parse(baseLayout)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, layoutData{
		ShopName: shopName,
		Content:  template.HTML(innerHTML),
	}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderInner(tmplStr string, data TemplateData) (string, error) {
	tmpl, err := template.New("inner").Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderEmail renders a full email for the given template type.
func RenderEmail(templateType string, data TemplateData) (string, error) {
	tmplStr, ok := templateMap[templateType]
	if !ok {
		tmplStr = genericTemplate
	}
	inner, err := renderInner(tmplStr, data)
	if err != nil {
		return "", err
	}
	return renderTemplate(data.ShopName, inner)
}

var templateMap = map[string]string{
	"order_confirmation":     orderConfirmationTemplate,
	"shipping_update":        shippingUpdateTemplate,
	"refund_notification":    refundNotificationTemplate,
	"abandoned_cart_reminder": abandonedCartReminderTemplate,
	"welcome_email":          welcomeEmailTemplate,
	"team_invitation":        teamInvitationTemplate,
}

const orderConfirmationTemplate = `<h2 style="margin:0 0 16px;">Order Confirmed</h2>
<p>Hi {{.CustomerName}},</p>
<p>Thank you for your order <strong>{{.OrderNumber}}</strong>. We have received your order and it is now being processed.</p>
<p>We will notify you once your order has shipped.</p>
<p style="margin-top:24px;">Thank you for shopping with us!</p>`

const shippingUpdateTemplate = `<h2 style="margin:0 0 16px;">Your Order Has Shipped</h2>
<p>Hi {{.CustomerName}},</p>
<p>Your order <strong>{{.OrderNumber}}</strong> is on its way!</p>
{{if .TrackingNumber}}<p><strong>Tracking Number:</strong> {{.TrackingNumber}}</p>{{end}}
{{if .TrackingURL}}<p><a href="{{.TrackingURL}}" style="color:#3182ce;text-decoration:underline;">Track your package</a></p>{{end}}
<p>You will receive another update when your package is delivered.</p>`

const refundNotificationTemplate = `<h2 style="margin:0 0 16px;">Refund Processed</h2>
<p>Hi {{.CustomerName}},</p>
<p>A refund of <strong>{{.RefundAmount}}</strong> has been processed for your order <strong>{{.OrderNumber}}</strong>.</p>
<p>Please allow 5-10 business days for the refund to appear on your statement.</p>
<p>If you have any questions, please don't hesitate to reach out.</p>`

const abandonedCartReminderTemplate = `<h2 style="margin:0 0 16px;">You Left Something Behind</h2>
<p>Hi {{.CustomerName}},</p>
<p>It looks like you left some items in your cart. Don't miss out!</p>
{{if .CartURL}}<p style="margin:24px 0;"><a href="{{.CartURL}}" style="display:inline-block;background:#3182ce;color:#ffffff;padding:12px 24px;border-radius:6px;text-decoration:none;font-weight:bold;">Complete Your Purchase</a></p>{{end}}
<p>If you have any questions about your items, we're happy to help.</p>`

const welcomeEmailTemplate = `<h2 style="margin:0 0 16px;">Welcome!</h2>
<p>Hi {{.CustomerName}},</p>
<p>Welcome to <strong>{{.ShopName}}</strong>! We're glad you're here.</p>
<p>Explore our latest products and discover something you'll love.</p>
<p style="margin-top:24px;">Happy shopping!</p>`

const teamInvitationTemplate = `<h2 style="margin:0 0 16px;">You've Been Invited</h2>
<p>Hi,</p>
<p><strong>{{.InviterName}}</strong> has invited you to join <strong>{{.ShopName}}</strong> as a <strong>{{.Role}}</strong>.</p>
{{if .InviteURL}}<p style="margin:24px 0;"><a href="{{.InviteURL}}" style="display:inline-block;background:#3182ce;color:#ffffff;padding:12px 24px;border-radius:6px;text-decoration:none;font-weight:bold;">Accept Invitation</a></p>{{end}}
<p>This invitation will expire in 7 days.</p>`

const genericTemplate = `<h2 style="margin:0 0 16px;">Notification</h2>
<p>{{.Body}}</p>`
