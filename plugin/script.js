browser.contextMenus.create({
  id: "download-with-sdm",
  title: "Download with SDM",
  contexts: ["link"],
});

browser.contextMenus.onClicked.addListener((info, tab) => {
  if (info.menuItemId === "download-with-sdm") {
    var gettingAllCookies = browser.cookies.getAll({
      url: info.linkUrl,
    }).then((cookies) => {
      var output = JSON.stringify({
        cookies: cookies,
        url: info.linkUrl,
        agent: navigator.userAgent,
      });
      var newURL = "web+sdm:" + encodeURIComponent(output);
      chrome.tabs.create({
        url: newURL,
      });
    });
  }
});
