'use strict';

(function(exports){

  function init(gmClient, contentElem, srcDir) {
    var gmClient = gmPageWrapper.createGMClient();
    var tagsInputElem = contentElem.find('#tags_input')
    var gmTagReqInst = gmTagRequester.create({
      tagsInputElem: tagsInputElem,
      allowNewTags: false,
      gmClient: gmClient,

      loadingStatus: function(isLoading) {
        if (isLoading) {
          contentElem.find("#tmploading").html("<p>...</p>");
        } else {
          contentElem.find("#tmploading").html("<p>&nbsp</p>");
        }
      },

      onChange: function(selectedTagIDs) {
        gmClient.getBookmarks(selectedTagIDs, function(bookmarks) {
          console.log('resp bkm2:', bookmarks)

          var listElem = contentElem.find("#tmpdata");
          listElem.text("");

          bookmarks.forEach(function(bkm) {
            var div = jQuery('<div/>', {
              id: 'bookmark_' + bkm.id,
              class: 'bookmark-div',
            });
            div.load(
              srcDir + "/bkm.html",
              undefined,
              function() {
                div.find("#bkm_link").html(bkm.title);
                div.find("#bkm_link").attr('href', bkm.url);
                div.find("#bkm_link").attr('target', '_blank');
                div.appendTo(listElem);
                console.log('html:', div.html());
              }
            );
          });

        })
      }
    });
  }

  exports.init = init;

})(typeof exports === 'undefined' ? this['gmGetBookmark']={} : exports);
