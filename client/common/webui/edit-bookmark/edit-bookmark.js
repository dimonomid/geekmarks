'use strict';

(function(exports){

  var contentElem = undefined;

  function init(gmClient, _contentElem, srcDir, queryParams) {
    contentElem = _contentElem;
    var tagsInputElem = contentElem.find('#tags_input')

    var gmTagReqInst = gmTagRequester.create({
      tagsInputElem: tagsInputElem,
      allowNewTags: true,
      gmClient: gmClient,

      loadingStatus: function(isLoading) {
        if (isLoading) {
          contentElem.find("#tmploading").html("<p>...</p>");
        } else {
          contentElem.find("#tmploading").html("<p>&nbsp</p>");
        }
      },
    });

    contentElem.find('#myform').on('submit', function(e) {
      //alert(gmTagReqInst.getSelectedTags);
      var selTags = gmTagReqInst.getSelectedTags();
      //alert(selTags.tagIDs);
      gmClient.saveBookmark(queryParams.bkm_id, {
        url: contentElem.find("#bkm_url").val(),
        title: contentElem.find("#bkm_title").val(),
        comment: contentElem.find("#bkm_comment").val(),
        tagIDs: selTags.tagIDs,
      }, function(status, resp) {
        if (status === 200) {
          console.log('saved', resp);
          window.close();
        } else {
          // TODO: show error
          alert(JSON.stringify(resp));
        }
      })
      return false;
    });

    gmClient.onConnected(true, function() {
      gmClient.getBookmarkByID(queryParams.bkm_id, function(status, resp) {
        console.log('getBookmarkByID resp:', status, resp);

        if (status === 200) {
          if (resp.url) {
            contentElem.find("#bkm_url").val(resp.url);
          }

          if (resp.title) {
            contentElem.find("#bkm_title").val(resp.title);
          }

          if (resp.comment) {
            contentElem.find("#bkm_comment").val(resp.comment);
          }

          resp.tags.forEach(function(tag) {
            gmTagReqInst.addTag(tag.id, tag.fullName, false)
          });
        } else {
          // TODO: show error
        }
      });
    });

  }

  exports.init = init;

})(typeof exports === 'undefined' ? this['gmEditBookmark']={} : exports);
