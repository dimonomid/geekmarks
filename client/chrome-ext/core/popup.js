$(document).ready(function() {
  $("#add_bookmark_link").click(function() {
    chrome.tabs.query({active: true, currentWindow: true}, function(arrayOfTabs) {
      var curTab = arrayOfTabs[0];
      var bg = chrome.extension.getBackgroundPage();
      bg.openPageAddBookmark(curTab);
    });
    return false;
  });

  $("#get_bookmark_link").click(function() {
    chrome.tabs.query({active: true, currentWindow: true}, function(arrayOfTabs) {
      var curTab = arrayOfTabs[0];
      var bg = chrome.extension.getBackgroundPage();
      bg.openOrRefocusPageWrapper("getBookmark", "page=get-bookmark", curTab);
    });
    return false;
  });
})

