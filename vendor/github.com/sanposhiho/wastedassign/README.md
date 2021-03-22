# wastedassign
`wastedassign` finds wasted assignment statements

found the value ...

- reassigned, but never used afterward
- reassigned, but reassigned without using the value

## Example

The comment on the right is what this tool reports

```
func f() int {
	a := 0 
        b := 0
        fmt.Print(a)
        fmt.Print(b)
        a = 1  // This reassignment is wasted, because never used afterwards. Wastedassign find this 

        b = 1  // This reassignment is wasted, because reassigned without use this value. Wastedassign find this 
        b = 2
        fmt.Print(b)
        
	return 1 + 2
}
```


## Installation

```
go get -u github.com/sanposhiho/wastedassign/cmd/wastedassign
```

## Usage

```
# in your project

go vet -vettool=`which wastedassign` ./...
```

# wastedassign(Japanese)
`wastedassign` は無駄な代入を発見してくれる静的解析ツールです。

以下のようなケースに役立ちます

- 無駄な代入文を省くことによる可読性アップ
- 無駄な再代入を検出することによる使用忘れの確認

また、使用しないことが明示的にわかることで、

- なぜ使用しないのか
- 使用しない変数が関数の返り値として存在した場合、関数の返り値として返す必要がないのではないか

などの議論が生まれるきっかけとなります。
