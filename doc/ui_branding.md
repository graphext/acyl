# UI Branding

For custom UI branding, pass the following JSON struct to the CLI:

```javascript
{
  "favicon_url": "https://someurl.com/favicon.png",
  "logo_url": "https://someurl.com/logo.png",
  // optional logo height/width in pixels (to scale image)  
  "logo_height": "48",
  "logo_width": "48",
  // optional additional styling for the logo (CSS)
  "logo_style": "padding-bottom: 3px",
  "logo_link": "https://someurl.com",
  "logo_alt": "Some Alt Text",
  // title is placed next to logo in the page header
  "title": "Some Title Text",
  // optional additional styling for the title text (CSS)
  "title_style": "font-weight: bold"
  // optional additional document urls for help view
  additional_doc_urls: {
    "CustomTitle": "https://myurl.com/docs"
    "CustomTitle2": "https://myotherurl.com/help"      
  }
}
```