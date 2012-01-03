package com.msrl.clocky;

import android.app.Activity;
import android.os.Bundle;
import android.webkit.WebView;

public class Clocky extends Activity
{
    WebView mWebView;

    /** Called when the activity is first created. */
    @Override
    public void onCreate(Bundle savedInstanceState) {
	super.onCreate(savedInstanceState);
	setContentView(R.layout.main);
	
	mWebView = (WebView) findViewById(R.id.webview);
	mWebView.getSettings().setJavaScriptEnabled(true);
	mWebView.loadUrl("http://10.0.1.13:8000");
    }
}
