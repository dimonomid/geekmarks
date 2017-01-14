$(document).ready(function() {

  var gmClientInst = gmClientFactory.create(true);

  $("#logo_div").html(gmLogo.getLogoDataHtml());

  applyUI();

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
    chrome.tabs.query({active: true, currentWindow: true}, function(arrayOfTabs) {
      var curTab = arrayOfTabs[0];
      var bg = chrome.extension.getBackgroundPage();
      bg.openOrRefocusPageWrapper("loginLogout", "page=login-logout", curTab);
      });

    return false;
  });

  $("#logout_link").click(function() {
    gmClientInst.logout().then(function() {
      applyUI();
    }).catch(function(e) {
      console.log('logout error:', e)
      alert('error:' + JSON.stringify(e));
    });

    return false;
  });

  // Show the login/logout box depending on whether the user is logged in now
  function applyUI() {
    gmClientInst.createGMClientLoggedIn().then(function(instance) {
      gmClientLoggedIn = instance;
      if (gmClientLoggedIn == null) {
        $('#logged_out_div').removeClass('hidden');
        $('#logged_in_div').addClass('hidden');
      } else {
        $('#logged_out_div').addClass('hidden');
        $('#logged_in_div').removeClass('hidden');
      }
    });
  }
})

