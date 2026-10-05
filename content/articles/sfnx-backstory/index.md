---
title: "AWS Step Functions用のコンパイラを作って気づいたこと"
date: 2026-10-06
images: ["ogp.png"]
tags: ["自作OSS"]
draft: false
---

Step Functions用のコンパイラを作ったら、いくつか気づいたことがありました。

<!--more-->

作ったのは[sfnx](https://github.com/iwamot/sfnx)というツールです。ツール自体の説明はZennの記事にまとめました。

- [Step FunctionsのワークフローをPythonで書ける「sfnx」を作った](https://zenn.dev/iwamot/articles/65bd0900005fd0)

以下、裏話として、その気づいたことを記します。

## コンパイラを作るのは楽しい

コンパイラ自作は、もともと楽しそうだと思っていましたが、予想以上に楽しい経験でした。

バランスを取らないといけない部分がいろいろあり、作者のセンスが問われるからです。

- ステートをどこまで畳み込むべきか（コンパイル速度とのトレードオフ）
- PythonとJSONataの挙動の違いをどこまで吸収すべきか（ASLの読みやすさとのトレードオフ）

現状のv3系では、ほどよいバランスになっていると自負しています。

## ツールを作るなら、AIユーザビリティテストが重要

v2系では、ステートの畳み込みや挙動の吸収にこだわることで、バランスが崩れていました。

たとえば、Pythonと同じくゼロ除算をエラーにするため、以下のようなJSONataを出力するなどです。

```jsonata
{% $count_val = 0 ? $error('division by zero') : $total / $count_val %}
```

崩れたバランスを整えるために考えたのが「AIユーザビリティテスト」でした。サブエージェントにsfnxを使うタスクを指示し、気になったところや、期待と異なる挙動を報告してもらう流れです。

このテストにおいて、上記のJSONataは「冗長」と指摘されました。サブエージェント自身がPython側でゼロをチェックしており、与えたタスクでは処理が重複していたのです。

報告された内容の修正とテストを繰り返すことで、バランスが自然に整っていきました。上記のJSONataは、v3系では以下のようにシンプルになっています。

```jsonata
{% $total / $count_val %}
```

このAIユーザビリティテスト、なかなか便利なので、[ai-usability-test](https://github.com/iwamot/skills/blob/main/skills/ai-usability-test/SKILL.md)スキルとして公開しました。よろしければお試しください。
