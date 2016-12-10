'use strict';

(function(exports){

  var contentElem = undefined;

  function init(gmClient, _contentElem, srcDir, queryParams, curTabData) {
    contentElem = _contentElem;
    var tagsTreeDiv = contentElem.find('#tags_tree_div')

    tagsTreeDiv.fancytree({
      source: [
        {title: "Node 1", key: "1"},
        {title: "Folder 2", key: "2", folder: true, children: [
          {title: "Node 2.1", key: "3"},
          {title: "Node 2.2", key: "4"}
        ]},
        {title: "Folder 3", key: "5", folder: true, children: [
          {title: "Node 3.1", key: "6"},
          {title: "Node 3.2", key: "7"}
        ]}
      ],
    });
  }

  exports.init = init;

})(typeof exports === 'undefined' ? this['gmTagsTree']={} : exports);
