/*
 Nginx search result modifier
 Using the js object "Search" from searchtools.js by Sphinx
 @Autor : oneTarek
*/
var nxSearchPage;

(function( $ ){
	nxSearchPage = {
		setInvRef : null,
		showControllerDocs : false,
		productNames : {
			"nginx" : "NGINX Plus",
			"nginx-plus" : "NGINX Plus",
			"nginx-ingress-controller" : "NGINX Ingress Controller",
			"nginx-app-protect" : "NGINX App Protect",
			"nginx-waf" : "NGINX WAF"
			

		},
		directoryNames : {
			"admin-guide" : "Admin Guide",
			"deployment-guides" : "Deployment Guides",
			"releases" : "Releases",
			"technical-specs" : "Technical Specifications"

		},
		init : function()
		{
			this.setInvRef = setInterval(this.checkSearchFinish, 500);
			this.addEvents();
			$(".search").hide();
		},

		addEvents : function()
		{
			$("#search-results").on('change', '#filter_by_product', function(e){
				//alert($(this).val());
				var p = $(this).val();
				if(p == 'all')
				{
					$(".search li").show();
				}
				else
				{
					$(".search li").hide();
					$(".product-" + p ).show();
				}
				//alert("click");
			});
		},

		procesSearchResult : function( hasResult )
		{//return false;
			clearInterval( nxSearchPage.setInvRef );
			if( hasResult )
			{
				var hlterms = nxSearchPage.getHighlightTerms();
				var items = Search.output.children('li');
				//alert(items.length);
				var i = 0;
				var n = 0;
				for( i=0; i<items.length; i++ )
				{
					var item = items[i];
					//alert(i);
					
					var A = $(item).children('a').first();
					var CONTEXT = $(item).children('.context').first();

					//alert(A.length);
					if( A.length )
					{
						//alert( A.length);
						
						var href = A.attr("href");
						//remove highlight parameter
						href = nxSearchPage.removeURLParameter( href , 'highlight' );
						A.attr('href', href ); 

						//take meta data from the list
						var pagename = href.replace("../", "");
						pagename = pagename.replace(/\/$/, "");//remove trailing slash
						var meta_data = {};
						if( typeof nx_meta_data_list != "undefined" )
						{
							if( pagename !="" &&  nx_meta_data_list.hasOwnProperty( pagename ) )
							{
								meta_data = nx_meta_data_list[ pagename ];
							}
						}


						//excludes
						if(href.indexOf("/home/") != -1 )
						{
							$(item).remove();
							continue;
						}
						//exclude main index file 
						if( pagename == "" )
						{
							$(item).remove();
							continue;
						}

						//Exclude tree pages. Page that has child page.
						if( meta_data.hasOwnProperty( "sourcename" ) )
						{
							var sourcename = meta_data['sourcename'];
							if( sourcename !="" && sourcename.indexOf("index.rst.txt") != -1 )
							{
								$(item).remove();
								continue;
							}
						}


						var info = nxSearchPage.getInfoFromUrl(href);
						var product = info['product_slug'];
						var directory = "";
						if( info['directory_slug'] != undefined )
						{
							directory = info['directory_slug'];
						}

						if( product != "")
						{
							A.after('<span class="search-item-info"><span class="product-name">'+info['product_name']+'</span><span class="directory-name">'+info['directory_name']+'</span></span>');
							$(item).addClass( "product-" + product );
							if( directory !="")
							{
								$(item).addClass( "directory-" + directory );
							}
							n++;
							
						}

						//show meta description in context
						if( CONTEXT.length )
						{
							var desc = "";
							if( meta_data.hasOwnProperty( "meta_description" ) )
							{
								desc = meta_data['meta_description'];
							}
							CONTEXT.html( desc );
							$.each(hlterms, function() {
						      CONTEXT = CONTEXT.highlightText(this, 'highlighted'); //highlightText() is defined in doctools.js
						    });

						}
						
					}//end if A.length
					

				}//end for
				
				//Search.status.text( "Search finished, found "+n+" page(s) matching the search query." );
				Search.status.text( "Search finished." );
				if( n!=0 )
				{
					//alert(n);
					//move technical-specs items to top
					
					var tec_specs_items = Search.output.children('.directory-technical-specs');
					
					for( i = tec_specs_items.length - 1; i>=0; i-- )
					{
						var itm = tec_specs_items[i];
						$(itm).prependTo(Search.output);
					}
					
					
					$( Search.output ).before( nxSearchPage.getFilterByProductHtml() );
				}
			}
			
			$(".search").show();
			
		},

		getInfoFromUrl : function( url )
		{
			var info = {
				'product_slug' : '',
				'product_name' : '',
				'directory_slug' : '',
				'directory_name' : ''
			}
			p = this.getProductSlugFromUrl( url );
			if( p !="")
			{
				info['product_slug'] = p;
				info['product_name'] = this.productNames[p];
				var parts = url.split("/"+p+"/");
				d = parts[1].split("/");
				info['directory_slug'] = d[0];
				info['directory_name'] = d[0];
				if( d[0] in this.directoryNames )
				{
					info['directory_name'] = this.directoryNames[ d[0] ];
				}
			}
			return info;

		},

		getProductSlugFromUrl : function(url)
		{
			var p = "";
			if(url.indexOf("/nginx/") != -1 )
			{
				p = "nginx";
			}
			else if(url.indexOf("/nginx-ingress-controller/") != -1 )
			{
				p = "nginx-ingress-controller";
			}
			else if(url.indexOf("/nginx-app-protect/") != -1 )
			{
				p = "nginx-app-protect";
			}
			else if(url.indexOf("/nginx-waf/") != -1 )
			{
				p = "nginx-waf";
			}
			return p;
		},

		getFilterByProductHtml : function()
		{
			var html = '<div class="filter-by-product-wrap">';
			html+="Filter result by product : ";
			html+='<select id="filter_by_product">';
				html+='<option value="all">All Products</option>';
				html+='<option value="nginx">NGINX Plus</option>';
				html+='<option value="nginx-ingress-controller">NGINX Ingress Controller</option>';
				html+='<option value="nginx-app-protect">NGINX App Protect</option>';
				html+='<option value="nginx-waf">NGINX WAF</option>';
				
			html+='</select>';
			html+="</div>";
			return html;
		},
		
		removeURLParameter : function(url, parameter)
		{
		    //prefer to use l.search if you have a location/link object
		    var urlparts= url.split('?');   
		    if (urlparts.length>=2) {

		        var prefix= encodeURIComponent(parameter)+'=';
		        var pars= urlparts[1].split(/[&;]/g);

		        //reverse iteration as may be destructive
		        for (var i= pars.length; i-- > 0;) {    
		            //idiom for string.startsWith
		            if (pars[i].lastIndexOf(prefix, 0) !== -1) {  
		                pars.splice(i, 1);
		            }
		        }

		        url= urlparts[0] + (pars.length > 0 ? '?' + pars.join('&') : "");
		        return url;
		    } else {
		        return url;
		    }
		},

		checkSearchFinish : function()
		{
			var text = $.trim( Search.status.text() );
			if( text!="" )
			{
				if( text.indexOf('Search finished') != -1 )
				{
					nxSearchPage.procesSearchResult( true );
				}
				else if( text.indexOf('Your search did not match any documents')!=-1 )
				{
					nxSearchPage.procesSearchResult( false );
				}
			}
		},

		getHighlightTerms : function ()
		{
			var hlterms = [];
			var params = $.getQueryParameters();
	      	if (params.q) {
	          	var query = params.q[0];

	          	var i;
			    var stopwords = ["a","and","are","as","at","be","but","by","for","if","in","into","is","it","near","no","not","of","on","or","such","that","the","their","then","there","these","they","this","to","was","will","with"];

			    var stemmer = new Stemmer();//defined in searchtools.js
			
			    var tmp = splitQuery(query);//defined in searchtools.js
			    for (i = 0; i < tmp.length; i++)
			    {
			      
			      if ($u.indexOf(stopwords, tmp[i].toLowerCase()) != -1 || tmp[i].match(/^\d+$/) || tmp[i] === "") 
			      {
			        // skip this "word"
			        continue;
			      }
			      // stem the word
			      var word = stemmer.stemWord(tmp[i].toLowerCase());
			      // prevent stemmer from cutting word smaller than two chars
			      if(word.length < 3 && tmp[i].length >= 3)
			      {
			        word = tmp[i];
			      }
			      
			      if (word[0] == '-') 
			      {
			        word = word.substr(1);
			      }
			      else 
			      {
			        hlterms.push(tmp[i].toLowerCase());
			      }
			      
			    }

	      	}//end if param.q
	      	return hlterms;
		}//end function 



	};//END OF nxSearchPage
	
	$(document).ready(function() {
	  nxSearchPage.init();
	});

})(jQuery);
