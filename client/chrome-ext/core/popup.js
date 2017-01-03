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

  $("#tags_tree_link").click(function() {
    chrome.tabs.query({active: true, currentWindow: true}, function(arrayOfTabs) {
      var curTab = arrayOfTabs[0];
      var bg = chrome.extension.getBackgroundPage();
      bg.openOrRefocusPageWrapper("tagsTree", "page=tags-tree", curTab);
    });

    return false;
  });

  $("#login_link").click(function() {
    gmClientInst = gmClient.create("localhost:4000");
    gmClientInst.login("google").then(function() {
      alert('logged in successfully');
    }).catch(function(e) {
      alert('error:' + JSON.stringify(e));
    });

    return false;
  });
})

