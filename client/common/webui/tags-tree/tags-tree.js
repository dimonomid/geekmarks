'use strict';

(function(exports){

  var contentElem = undefined;

  function init(gmClient, _contentElem, srcDir, queryParams, curTabData) {
    contentElem = _contentElem;
    var tagsTreeDiv = contentElem.find('#tags_tree_div')

    gmClient.getTagsTree(function(status, resp) {
      if (status == 200) {
        var treeData = convertTreeData(resp);

        tagsTreeDiv.fancytree({
          source: treeData.children,
        });
      } else {
        // TODO: show error
        alert(JSON.stringify(resp));
      }
    })
  }

  function convertTreeData(tagsTree) {
    var ret = {
      title: tagsTree.names.join(","),
      key: tagsTree.id,
    };
    if ("subtags" in tagsTree) {
      ret.children = tagsTree.subtags.map(function(a) {
        return convertTreeData(a);
      });
      //ret.folder = true;
    }
    return ret;
  }

  exports.init = init;

})(typeof exports === 'undefined' ? this['gmTagsTree']={} : exports);
