(function( $ ){
	
	function scrollToHash( on_page_load ){

		if( window.location.hash == undefined || window.location.hash == ""){ return ;}
		var hash_text = window.location.hash.replace("#", "");
		var scroll_h = $("#nx_masthead").height() + 15;
		
		if( on_page_load == true )
		{
			var elm = false;
			if( $("#"+hash_text).length )
			{
				elm = $("#"+hash_text);
			}
			else if( $("[name='"+hash_text+"']").length )
			{
				elm = $("[name='"+hash_text+"']");
			}		
			if( elm != false )
			{
				$('html, body').animate({
		           'scrollTop':   $(elm).offset().top - scroll_h
		         }, 1000);
			}
		}
		else
		{
			window.scrollBy(0, -scroll_h);
		}
		
	}//end function

	$(document).ready(function(){
	    
	    /*
	    var controller_doc_link = $('.nginx-doc-sidenav li a[href="https://www.nginx.com/products/nginx-controller/"]');
		if( controller_doc_link.length )
		{
			$(controller_doc_link).attr("href", "/");
			$(controller_doc_link).parent().addClass('has-sub-nav');
			$(controller_doc_link).parent().append('<ul class="sub-side-nav"><li><a href="/"><span style="font-size:18px;color:#898787;font-weight: 300;">Available soon</span></a></li></ul>');
		}
		*/
	    $('.nginx-doc-sidenav li').each(function(){
			if($(this).find('ul').length>0){
				$(this).prepend("<span class='open-sidenav'></span>");
				$(this).addClass('has-sub-nav');
				$(this).find('ul').addClass('sub-side-nav');
			}
			if($(this).hasClass('current')){
				$(this).children('ul').css('display','block');
			}
		});
		
	    $('.nginx-doc-sidenav .open-sidenav').on('click',function(){
	      $(this).parent('li').toggleClass('current');
	      $(this).siblings('.sub-side-nav').slideToggle(100);
	      return false;
	    });

	    //when I click on an item on the TOC that has a dropdown, instead of going to the table of contents page, it should open the dropdown.
	    $('.nginx-doc-sidenav a').on('click',function(){
	      
	      
	      var sub_menu = $(this).next('.sub-side-nav');
	      if( sub_menu.length )
	      {
	      	$(this).parent('li').toggleClass('current');
	      	$(this).siblings('.sub-side-nav').slideToggle(100);
	      	return false;

	      	/*
	      	sub_menu_first_link = $(sub_menu).children('li').first().children('a').first();

	      	if(sub_menu_first_link.length)
	      	{
	      		
	      		var this_is_current = $(this).hasClass('current');
	      		var fl_is_current = $(sub_menu_first_link).hasClass('current');
	      		var fl_has_hash = ($(sub_menu_first_link).attr('href').indexOf("#") != -1) ? true : false;

	      		if( fl_is_current || ( !fl_is_current  && !fl_has_hash ) )
	      		{
	      			$(this).parent('li').toggleClass('current');
	      			$(this).siblings('.sub-side-nav').slideToggle(100);
	      			return false;
	      		}
	      		else if( this_is_current )
	      		{
	      			//alert("current");
	      			$(this).parent('li').toggleClass('current');
	      			$(this).siblings('.sub-side-nav').slideToggle(100);
	      		}
	      	}
	      	*/
	      }
	      
	      if( $(this).attr('href') == "/" )
	      {
	      	return false;
	      }

	      return true;
	      
	    });

		$(".code-terminal").parent().next(".highlight-default").addClass("terminal");

		// Doc responsive menu code
		var _w = $(window).width();
		var searchContent = $('.nginx-doc-sidebar-inner .search-doc-wrapper').html();
		$('.nginx-doc-sidebar').prepend('<div class="search-doc-wrapper mob-search-doc">'+searchContent+'</div><div class="doc-mob-sidenav"><h3>NGINX Documentation<span class="icon icon-arrow-down"></span></h3></div>');

		$('.nginx-doc-sidebar').on('click',".doc-mob-sidenav",function(){
			$('.nginx-doc-sidebar-inner').slideToggle(300);
			$('.doc-mob-sidenav span').toggleClass('icon-arrow-up');
		});

		// Search box fix issue
		var $search_wrap = $('.search-doc-wrapper');
		var $doc_side_inner = $('.nginx-doc-sidebar-inner');
		function search_box_size(){
			$search_wrap.css('width',($doc_side_inner.width()-20)+'px');
		}
		if(_w >= 800){search_box_size();}		
		$(window).resize(function() {
			_w = $(window).width();
			if(_w >= 800){search_box_size();}
			else{
				$search_wrap.css('width','100%');
			}
		});
		
		// doc sidebar fixed
		var sidebarHeight = $(window).height() - $('#nx_masthead').height();
		function sidebar_height(){
			if(_w >= 800){
				$('.nginx-doc-sidebar').css('height',sidebarHeight+'px');
				$('.nginx-doc-sidebar-inner').css('height', (sidebarHeight - 70)+'px');
			}
		}
		
		sidebar_height();

		$(window).scroll(function() {
			var footer_top = $('.pre-footer-wrap').offset().top;
			var window_height = $(window).height();
		    var cur_pos = $(window).scrollTop();
		    var cur_bottom_pos = cur_pos + window_height;

		    if(cur_bottom_pos >= footer_top){
		    	$('.nginx-doc-sidebar').addClass('remove-sticky-sidebar');
		    	$('.nginx-doc-sidebar-inner,.nginx-doc-sidebar').css('height', 'auto');
		    }else{
		    	$('.nginx-doc-sidebar').removeClass('remove-sticky-sidebar');
		    	sidebar_height();
		    }
		    
		});

		if (!!window.location.hash)
		{
			scrollToHash( true );
		}
		$(window).on('hashchange', function() {
	      scrollToHash( false );
		});

	});//end document ready

})(jQuery);


/*
 * Jump to appropriate section of page for wrong fragment parameters (#hash-tag)
 * Rewriting anchor tags
 */
var hash_tag_pairs = {
	'#wrong-place' : '#correct-place'
	//add more items here 
};

if(window.location.hash != '')
{
	for( i in hash_tag_pairs )
	{ 
		if( window.location.hash == i )
		{
			window.location.hash = hash_tag_pairs[i];
			break;
		}
	}
}
