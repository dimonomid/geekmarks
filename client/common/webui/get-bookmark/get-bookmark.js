'use strict';

(function(exports){

  var gmClientLoggedIn = undefined;
  var gmTagReqInst = undefined;
  var loadingSpinner = undefined;

  function init(_gmClient, contentElem, srcDir) {
    _gmClient.createGMClientLoggedIn().then(function(instance) {
      loadingSpinner = contentElem.find('#loading_spinner');
      if (instance) {
        initLoggedIn(instance, contentElem, srcDir);
      } else {
        gmPageWrapper.openPageLogin("openPageGetBookmark", []);
        gmPageWrapper.closeCurrentWindow();
      }
    });
  }

  function initLoggedIn(instance, contentElem, srcDir) {
    gmClientLoggedIn = instance;
    var tagsInputElem = contentElem.find('#tags_input')
    gmTagReqInst = gmTagRequester.create({
      tagsInputElem: tagsInputElem,
      allowNewTags: false,
      gmClientLoggedIn: gmClientLoggedIn,

      loadingStatus: function(isLoading) {
        // TODO: add a separate spinner for tags loading
        /*
        if (isLoading) {
          loadingSpinner.show();
        } else {
          loadingSpinner.hide();
        }
        */
      },

      onChange: function(selectedTags) {
        requestBookmarks(selectedTags.tagIDs);
      }
    });

    function requestBookmarks(tagIDs) {
      loadingSpinner.show();
      gmClientLoggedIn.getTaggedBookmarks(
        tagIDs,
        function(status, resp) {
          loadingSpinner.hide();
          onBookmarksReceived(status, resp);
        }
      );
    }

    gmClientLoggedIn.onConnected(true, function() {
      requestBookmarks([]);
    });

    function onBookmarksReceived(status, resp) {
      if (status === 200) {
        var listElem = contentElem.find("#bookmarks_list");
        listElem.text("");

        resp.forEach(function(bkm) {
          var div = jQuery('<div/>', {
            id: 'bookmark_' + bkm.id,
            class: 'bookmark-div',
          });
          div.load(
            srcDir + "/bkm.html",
            undefined,
            function() {
              var uriHost = new URI(bkm.url).host();
              var faviconTag = uriHost ? "<img src='https://www.google.com/s2/favicons?domain=" + encodeURIComponent(uriHost) + "' />" : "";
              var $bkmLink = div.find("#bkm_link");
              var $control = div.find("#control");
              $bkmLink.html(faviconTag + " " + bkm.title);
              $bkmLink.attr('href', bkm.url);
              $bkmLink.attr('target', '_blank');

              var $tagsP = div.find("#tags");
              var tags = bkm.tags || []
              $tagsP.html(tags.map(function(tag) {
                return "/" + tag.items.map(function(tagItem) {
                  return tagItem.name || "";
                }).join("/");
              }).join(", "));

              // Just after user clicked at some bookmark, close the
              // bookmark selection window
              $bkmLink.click(function() {
                window.close();
                return true;
              })

              div.find("#edit_link").click(function() {
                gmPageWrapper.openPageEditBookmarks(bkm.id);
                return false;
              })

              div.find("#del_link").click(function() {
                if (confirm("Delete this bookmark?")) {
                  gmClientLoggedIn.deleteBookmark(bkm.id, function(status, resp) {
                    if (status === 200) {
                      requestBookmarks(gmTagReqInst.getSelectedTags().tagIDs);
                    } else {
                      // TODO: show error
                      alert(JSON.stringify(resp));
                    }
                  });
                }
                return false;
              })

              div.mouseover(function() {
                $control.show();
              });

              div.mouseout(function() {
                $control.hide();
              });

              div.appendTo(listElem);
            }
          );
        });
      } else {
        // TODO: show error
        alert(JSON.stringify(resp));
      }
    }
  }

  exports.init = init;

})(typeof exports === 'undefined' ? this['gmGetBookmark']={} : exports);
