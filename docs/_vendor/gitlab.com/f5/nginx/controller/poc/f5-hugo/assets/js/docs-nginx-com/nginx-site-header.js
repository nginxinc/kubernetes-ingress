/**
 * nginx-site-header.js
 *
 * All JS related to site header area
 */

 (function($){

    var nginxSiteHeader = {

        /**
         * Expand/collapse the header for mobile.
         */
        mobileHeaderToggle: function() {
            var expandedClass = 'mobile-expanded',
                $header = $('#nx_masthead');

            $header.stop(true, true);
            if (!$header.hasClass(expandedClass)) {
                $header.addClass(expandedClass);
            } else {
                $header.removeClass(expandedClass);
            }
        },

        /**
         * Toggle expansion of a primary menu item that has children (mobile display only).
         *
         * @param triggerElement
         */
        mobilePrimaryMenuItemToggle: function(triggerElement) {
            var expandedClass = 'item-mobile-expanded',
                $item = $(triggerElement).closest('.menu-item'),
                $iconElement = $item.find('.icon').first();

            $item.stop(true, true);
            if (!$item.hasClass(expandedClass)) {
                $item.addClass(expandedClass);
                $iconElement.removeClass('icon-arrow-down').addClass('icon-arrow-up');
            } else {
                $item.removeClass(expandedClass);
                $iconElement.removeClass('icon-arrow-up').addClass('icon-arrow-down');
            }
        }

    };//end nginxSiteHeader

    $(document).ready(function(){

        // Toggle mobile header.
        $('.mobile-menu-button').on('click', function(e) {
            e.preventDefault();
            nginxSiteHeader.mobileHeaderToggle();
        });

        // Toggle mobile menu item.
        $('body').on('click', '#nx_masthead.mobile-expanded #menu-primary > li.menu-item-has-children > a .icon', function(e) {
            e.preventDefault();
            nginxSiteHeader.mobilePrimaryMenuItemToggle(this);
        });

        // Toggle menu search
        $('.search-field')
            .on('focus', function() {
                $(this).closest('.search-form-wrapper').addClass('focused');
                $('.nx-site-title').addClass('logo-left');//mobile view max-width: 600px
                if(!$("#nx_masthead").hasClass("sticky-menu")) {
                    $('.header-extras-wrapper').not('.mobile-expanded .header-extras-wrapper').toggle();
                }
            })
            .on('blur', function() {
                $(this).closest('.search-form-wrapper').removeClass('focused');
                $('.nx-site-title').removeClass('logo-left');//mobile view max-width: 600px
                if(!$("#nx_masthead").hasClass("sticky-menu")) {
                    setTimeout(function () {
                        $('.header-extras-wrapper').not('.mobile-expanded .header-extras-wrapper').fadeToggle();
                    }, 1000);
                }
            });
        //end toggle menu search

        // Navbar scroll function
        var w = $(window).width();
        var h = $(window).height()-69;
        if(w <= 979){
            $('.nx-header-menus').css('max-height',h+'px');
        }

    });//end document ready

 })(jQuery);