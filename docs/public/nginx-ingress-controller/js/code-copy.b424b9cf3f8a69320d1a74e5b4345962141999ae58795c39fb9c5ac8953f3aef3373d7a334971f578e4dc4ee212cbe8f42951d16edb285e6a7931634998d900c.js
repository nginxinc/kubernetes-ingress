function CopyCode(clipboard) {
    document.querySelectorAll('pre > code').forEach(function(codeBlock) {
        var button = document.createElement('button');
        button.className = 'code-copy';
        button.type = 'button';
        button.innerHTML = '<i class="fas fa-copy"></i> Copy';

        button.addEventListener('click', function() {
            // removes return carriage from copied text
            clipboard.writeText(codeBlock.textContent.replace(/\n$/g, "")).then(
                function() {
                    button.blur(); /* Chrome fix */
                    button.innerHTML = '<i class="fas fa-check"></i> Copied!';
                    setTimeout(function() {
                        button.innerHTML = '<i class="fas fa-copy"></i> Copy';
                    }, 2000);
                },
                function(error) {
                    button.innerHTML = '<i class="fas fa-exclamation"></i> Error';
                    console.error(error);
                }
            );
        });

        var pre = codeBlock.parentNode;
        if (pre.parentNode.classList.contains('highlight')) {
            var highlight = pre.parentNode;
            highlight.parentNode.insertBefore(button, highlight);
        }
    });
}

CopyCode(navigator.clipboard);
