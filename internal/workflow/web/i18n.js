'use strict';

// English source phrases are the catalog keys. Values never change API fields.
const CommonReelI18n = (() => {
 const zh = {
  'CommonReel · Good films. Better company.':'CommonReel · 好电影，一起选',
  'Skip to content':'跳到主要内容','CommonReel home':'CommonReel 首页',
  'GOOD FILMS. BETTER COMPANY.':'好电影，一起选','Good films. Better company.':'好电影，一起选',
  'Starting up':'正在启动','Local app ready':'应用已就绪','Room connected':'房间已连接','Reconnecting…':'正在重新连接…',
  'Open reusable profile':'打开个人资料','Language':'界面语言',
  'YOUR NEXT MOVIE NIGHT':'今晚，一起看部电影','Less deciding.':'少点纠结，','More':'多点','watching.':'观影。',
  'Different tastes. One couch. Find a film that fits your whole room, without the endless scroll.':'口味不同，也能一起选。把今晚的时间、平台和偏好放在一起，找到适合大家的电影。',
  'Give your movie night a name':'给今晚的电影局起个名字','Friday at ours':'比如：周五客厅电影局','Create a room':'创建房间',
  '2–4 friends':'2–4 位朋友','Same Wi-Fi':'同一 Wi-Fi','No account needed':'无需注册',
  'THE ADVENTURE':'去冒险','Somewhere':'去个','new.':'新地方。','LEAVE THE EVERYDAY BEHIND':'暂时告别日常',
  'A COMMONREEL ORIGINAL DESIGN':'COMMONREEL 原创设计','One night.':'一个夜晚，','Good company.':'几个好友。','THERE’S A FILM FOR THIS ROOM.':'总有一部，适合今晚的我们。',
  'A little less “you choose.”':'少一点「随便，你选吧」。','SMALL LIBRARY. CLEAR LIMITS.':'精选片库，条件清楚',
  'A verified':'可核对的','film collection.':'电影片库。','US subscription availability checked when you pick.':'选片时核对美国地区的订阅供应。',
  'Bring your people':'叫上朋友','One room. One invitation link.':'一个房间，一条邀请链接。','Set tonight’s limits':'设好今晚的条件','Your platforms, time and no-gos.':'平台、时长，还有不想看的类型。',
  'Make a shared choice':'记录小组选择','A short list. A clear group record.':'几部候选，一份清楚的选择记录。',
  'Movie night navigation':'观影步骤','YOUR MOVIE NIGHT':'今晚的电影局','Steps':'步骤','Set up the room':'设置房间','Room':'房间','Find your film':'挑选电影','Films':'电影','The group choice':'小组选择','Choice':'选择',
  'A LITTLE HELP':'需要一点帮助？','Movie night assistant':'选片助手','Your reusable profile':'可复用的个人资料','About the film library':'了解片库',
  'Made for the same couch.':'今晚，坐到一起。','Share the invite with friends on your Wi-Fi.':'把邀请链接发给同一 Wi-Fi 下的朋友。','End this movie night':'结束本场观影',
  'Your room':'你的房间','Set up':'房间设置','Open MCP movie night assistant':'打开 MCP 选片助手','Assistant · MCP':'助手 · MCP','Invite friends':'邀请朋友',
  'Bring your friends in to get started.':'先邀请朋友加入。','Edit limits':'修改条件',
  'STEP 01 / THE PEOPLE & THE PLAN':'第 01 步 / 朋友与安排','Make it your movie night.':'今晚的电影，一起安排。','A few quick details, then we’ll find your shortlist.':'填好几个条件，就能开始挑选。',
  'Who’s watching?':'今晚谁来看？','Two to four people. Different tastes welcome.':'2–4 人，欢迎不同口味。','Your nickname':'你的昵称','Join room':'加入房间',
  'Everyone with the invite can edit or record for any member.':'持邀请链接的人均可替成员修改或记录。',
  'Tonight’s ground rules':'今晚的选片条件','What works on the screen you’ll use?':'按今晚实际使用的设备和账号选择。','How much time do you have?':'今晚有多少时间？','minutes':'分钟','Runtime presets':'常用时长','90m':'90 分','2h':'2 小时','2½h':'2.5 小时',
  'Your streaming services':'可用的流媒体平台','United States':'美国地区','Rewatching is okay':'可以重看','Include films someone has already seen.':'允许推荐成员已看过的电影。','Save tonight’s limits':'保存今晚的条件',
  'Subscription listings only. Your platform account and add-ons still need to be checked.':'仅核对平台的订阅供应，账号权益及附加套餐需自行确认。',
  'Bring at least two people into the room.':'至少两人加入后即可开始。','Then save your time limit and streaming services.':'然后保存时长和可用平台。','Continue to films':'开始挑选电影',
  'STEP 02 / LESS SCROLLING, MORE POSSIBILITY':'第 02 步 / 少翻几页，多点可能','Find your kind of tonight.':'找到适合今晚的那一部。','A short list that respects everyone’s recorded limits.':'候选影片会满足已记录的每个人的限制。',
  'Find films that fit':'查找符合条件的电影','Sort by':'排序方式','Balance the room':'兼顾全组','Average estimate':'平均估计评分','Catalog baseline':'片库基础排序',
  'YOUR NEXT':'今晚的','GOOD FILM.':'好电影。','Your shortlist starts here.':'从这里开始挑选。','We’ll check the library against your time, platforms, vetoes and known watched films.':'按时长、平台、类型否决和已知观看记录筛选片库。','Review the room’s limits':'查看并调整房间条件','What was filtered out?':'哪些影片被排除了？',
  'Original typographic cover art. Scores are estimates, not personal acceptance.':'原创文字封面。评分是估计值，接受情况以成员记录为准。','Availability by Watchmode ↗':'供应数据来自 Watchmode ↗',
  'Record the group choice':'记录小组选择','STEP 03 / MAKE THE CALL':'第 03 步 / 做个选择','One film. A shared record.':'一部电影，一份共同记录。','Record a response for each member, then save the choice.':'记录每个成员的表态，再保存选择。',
  'Found one you like?':'有中意的电影了？','Propose a film from the shortlist to start the room’s response record.':'从候选中提议一部电影，开始记录大家的表态。','Explore your shortlist':'看看候选电影','The room’s responses':'成员表态','Save group-recorded choice':'保存小组记录的选择',
  'Anyone can record for someone else. These records do not prove personal consent. Saving checks supply again.':'允许代填，记录不证明本人同意。保存时会再次核对供应。',
  'Assistant':'助手','Your profile':'个人资料','End room':'结束房间','217-film library · Data & sources ↗':'217 部电影 · 数据与来源 ↗',
  'A small room. A shared choice.':'一个小房间，一起做选择。','Built for movie night, together.':'为一起看电影而做。',
  'TONIGHT’S PREFERENCES':'今晚的观影偏好','Member preferences':'成员偏好','Close member preferences':'关闭成员偏好',
  'SAVE THEM A SEAT':'给朋友留个位置','Movie night is better together.':'好电影，和朋友一起看。','Close invitation':'关闭邀请',
  'Scan or open this link on the same Wi-Fi. Everyone holding it can edit this room.':'在同一 Wi-Fi 下扫码或打开链接。持链接的人均可编辑房间。','Your invitation link':'邀请链接','Copy invitation link':'复制邀请链接','Use the computer’s LAN address for phones. A 127.0.0.1 link only opens on this computer.':'手机请使用电脑的局域网地址。127.0.0.1 链接仅适用于这台电脑。',
  'A LITTLE HELP DECIDING':'让选片轻松一点','Talk movie night.':'和助手聊聊今晚。','Close assistant':'关闭助手',
  'Alexa+ Web Simulation · MCP Streamable HTTP 2025-11-25 · 7 tools':'Alexa+ 网页模拟 · MCP Streamable HTTP 2025-11-25 · 7 项工具',
  'Change an explicit limit, find films or ask why one fits. You can keep using the room while the assistant thinks.':'修改明确的条件、查找电影，或问问推荐理由。助手思考时，你仍可操作房间。',
  'Find a film for us ↗':'帮我们选部电影 ↗','Explain our limits ↗':'解释当前条件 ↗','Find films that fit our limits and explain the first candidate.':'请查找符合我们条件的电影，并解释第一部候选为什么合适。','Explain the room’s current movie-night limits.':'请解释房间当前的观影条件。',
  'Who is speaking?':'谁在说话？','Unspecified':'未指定','Your message':'你想说什么？','For example: Alex does not want Horror tonight.':'例如：Alex 今晚不想看恐怖片。','Ask the local assistant':'询问本地助手','Runs on your computer. If it’s unavailable, all room controls still work.':'助手在本机运行。暂不可用时，仍可通过页面操作。',
  'FOR THIS NIGHT & THE NEXT':'今晚用，下次也能用','Close profile':'关闭个人资料','Keep normalized viewing history and stable preferences. Applying a profile makes a separate copy for tonight.':'保存整理后的观看历史与长期偏好。应用到房间时，会生成当晚的独立副本。',
  'Create a profile':'创建个人资料','or load one you saved':'或载入已有资料','Profile recovery code':'资料恢复码','Load profile':'载入资料','Stable liked genres':'长期喜欢的类型','Comedy, Drama':'Comedy, Drama（喜剧、剧情；可点选下方类型）','Save stable preferences':'保存长期偏好',
  'Import Netflix viewing history':'导入 Netflix 观看历史','Netflix Viewing Activity CSV':'Netflix 观看活动 CSV','Replace previous imported history only':'仅替换此前导入的历史','Import into this profile':'导入到这份资料','Delete profile and withdraw active copies':'删除资料并撤回活动场次副本','Room edits never write back automatically. Keep your recovery code private.':'房间修改不会自动写回长期资料。请妥善保管恢复码。',
  'YOUR VIEWING INPUTS':'你的观看记录','Known watched & pending titles':'已知看过与待确认影片','Close viewing history':'关闭观看记录',
  'THE DETAILS BEHIND THE PICKS':'推荐背后的数据','A small library. Honest data.':'小片库，数据说清楚。','Close data information':'关闭数据信息','Limits come first.':'限制优先。','Unknown runtime or subscription supply stays excluded. Nothing is silently relaxed.':'时长或订阅供应未知的影片会被排除，条件不会自动放宽。','Watched doesn’t mean liked.':'看过，不等于喜欢。','Scores estimate preferences. Unmatched viewing history does not mean never watched.':'评分用于估计偏好。历史未匹配成功，不代表从未看过。','Streaming availability powered by Watchmode ↗':'流媒体供应数据来自 Watchmode ↗','Movie metadata: Wikidata · CC0. US subscription listings do not verify your account or add-ons. No playback is provided. Cover illustrations are original typographic designs.':'影片元数据：Wikidata · CC0。美国订阅供应不代表个人账号或附加套餐权益。本站不提供播放，封面为原创文字设计。',
  'Choice saved':'选择已保存','Checking current subscription supply…':'正在核对当前订阅供应…','Choice saved for these limits.':'已按当前条件保存选择。','All member entries record acceptance. Ready to save.':'所有成员条目均已记录接受，可以保存。','{accepted}/{total} accepted':'{accepted}/{total} 已记录接受','Waiting for {count} more acceptance records.':'还需 {count} 条接受记录。',
  'supply not confirmed':'供应未确认','no selected subscription listing':'所选平台无订阅供应','over the runtime limit':'超过时长限制','a genre veto':'命中类型否决','known watched':'已知看过','metadata not confirmed':'影片资料未确认','room or limits incomplete':'成员或条件不完整',
  '{count} {reason}':'{count} 部：{reason}','rows':'行','matched':'已匹配','duplicates':'重复','known episodes':'已知剧集','ambiguous':'匹配有歧义','unmatched':'未匹配','invalid':'无效','{count} {label}':'{label} {count}',
  'Comedy':'喜剧','Drama':'剧情','Action':'动作','Thriller':'惊悚','Horror':'恐怖','Romance':'爱情','Sci-Fi':'科幻','Animation':'动画','Adventure':'冒险','Crime':'犯罪','Mystery':'悬疑','Fantasy':'奇幻','Family':'家庭','Documentary':'纪录片','Music':'音乐','War':'战争','Western':'西部','History':'历史','Biography':'传记',
  '{name}’s preferences':'{name} 的偏好','These are tonight’s inputs. Any member may edit this record.':'这是今晚的偏好，房间内任何人均可修改。','Types of film they enjoy':'喜欢的电影类型','Types to skip tonight':'今晚不想看的类型','Save member preferences':'保存成员偏好','Watched films & ratings':'观看历史与评分','Apply loaded profile':'应用已载入的资料','Load a reusable profile':'载入可复用资料','Optional: import Netflix viewing history':'可选：导入 Netflix 观看历史','Replace imported history only':'仅替换导入的历史','Import for this member':'为该成员导入','Reliable movie matches become known watched. Unmatched titles stay pending; watched never means liked.':'可靠匹配的电影记为已看。未匹配标题保留待确认；观看记录不作为喜欢的证据。','Remove {name} from this room':'将 {name} 移出房间',
  'COMMONREEL SELECTS':'COMMONREEL 精选','FILM NIGHT':'电影之夜','A FILM FOR THIS ROOM':'适合这一屋人的电影',
  '{total} films in the library · {verified} supply-verified · {unknown} with unknown supply excluded':'片库 {total} 部 · 供应已确认 {verified} 部 · 排除供应未知 {unknown} 部',
  '{reasons}. Up to {limit} new supply checks per round. No recorded limits are relaxed.':'{reasons}。每轮最多新增核对 {limit} 部影片的供应，不放宽已记录条件。','No films meet these limits yet.':'暂时没有符合条件的电影。','Review the exclusions below. Change limits explicitly, or check supply again; unknown information stays excluded.':'查看下方排除原因后，明确修改条件或重新查询供应；信息未知的影片仍会排除。',
  '{count} min':'{count} 分钟','Estimated group fit':'小组匹配估计','Provider {id}':'平台 {id}','Why it fits the room':'为什么适合大家','{name} · estimated {score} / 5':'{name} · 估计 {score} / 5','Within the saved runtime, veto and rewatch limits. Subscription supply is confirmed for a selected service.':'符合已保存的时长、类型否决与重看条件，且已确认至少一个所选平台提供订阅供应。','Propose this film':'提议这部电影','US subscription · checked {date}':'美国订阅供应 · 核对于 {date}',
  'Shared editing · Ends {date}':'共享编辑 · {date} 到期','No genre vetoes':'未设置类型否决','Skip {genres}':'跳过 {genres}','{vetoes} · {count} watched':'{vetoes} · 已看 {count} 部','Edit':'编辑','Edit {name} preferences':'编辑 {name} 的偏好','Save a seat for a friend':'给朋友留个位置','Your movie night starts with you':'由你开始今晚的电影局',
  '{count} people · {runtime} · {services}':'{count} 人 · {runtime} · {services}','Time not set':'未设置时长','Choose your services':'选择可用平台','The room is ready for a shortlist.':'准备好了，可以挑选电影。','Choose a service and save tonight’s limits.':'选择平台并保存今晚的条件。','You can edit individual tastes with the Edit buttons.':'点击「编辑」可调整各成员的偏好。','Your shortlist uses saved limits, not unsaved changes.':'候选筛选使用已保存的条件。','{count} min max':'最长 {count} 分钟','Set runtime':'设置时长','Rewatch okay':'允许重看','No known rewatches':'排除已知看过的影片','Edit tonight’s limits':'编辑今晚的条件',
  'The room’s inputs changed. Find films again to refresh your shortlist.':'房间条件已变更，请重新查找候选。','Add at least two people, choose your services, and save the limits in Set up the room.':'请先在房间设置中加入至少两位成员，选择平台并保存条件。','Saved group choice':'已保存的小组选择','Earlier choice, under previous limits':'此前条件下保存的选择','{label}: {title}. This does not mark the film as watched.':'{label}：{title}。保存选择不会将影片标记为已看。','{year} · {count} minutes':'{year} · {count} 分钟','Proposed for the room. Saving checks subscription supply again.':'已向房间提议。保存时会再次核对订阅供应。','Recording responses for {title}.':'正在记录对《{title}》的表态。','Acceptance recorded':'已记录接受','Rejection recorded':'已记录反对','Waiting for a response':'等待表态','Accept':'接受','Reject':'反对','Record accept for {name}':'记录 {name} 接受','Record reject for {name}':'记录 {name} 反对',
  'Checking films…':'正在查询电影…','Checking a bounded set of US subscription listings. You can keep editing the room.':'正在核对部分影片的美国订阅供应，你仍可编辑房间。','{count} films fit the recorded limits. Pick one to propose to the room.':'找到 {count} 部符合已记录条件的电影，可以提议其中一部。','No qualifying films found. Review the exclusions and adjust limits explicitly.':'未找到符合条件的电影，请查看排除原因并明确调整条件。','Could not finish the shortlist. {reason}':'候选查询未完成。{reason}',
  'Previous 100':'上一页 100 条','Next 100':'下一页 100 条','Loading film title…':'正在加载片名…','External movie · IMDb {id}':'库外电影 · IMDb {id}','Film outside the current library':'当前片库以外的电影','Film title unavailable':'片名暂不可用','{seen} known watched · {pending} pending. Showing up to 100 of each.':'已知看过 {seen} 部 · 待确认 {pending} 条，每类每页最多 100 条。','Latest import: {report}':'最近导入：{report}',
  'More than one possible match. Choose the correct film.':'有多个可能匹配，请选择正确的电影。','No reliable match yet. Confirm a film below.':'尚无可靠匹配，请在下方确认电影。','Search the film library':'搜索片库','Find a film for {title}':'为 {title} 查找匹配影片','Film matching {title}':'与 {title} 匹配的影片','Choose an explicit film':'明确选择一部电影','Search':'搜索','Choose a film':'选择电影','Confirm watched film':'确认已看影片','IMDb movie ID, e.g. tt0114709':'IMDb 电影编号，例如 tt0114709','External IMDb movie ID for {title}':'{title} 对应的 IMDb 电影编号','Confirm external movie ID · history only':'确认库外电影编号 · 仅用于历史',
  'Add explicit watched / rating / like':'手动记录已看、评分或喜欢','Search movie title':'搜索电影名称','Search a movie to record watched, rated or liked':'搜索电影并记录已看、评分或喜欢','Movie for explicit input':'要记录的电影','Search movies':'搜索电影','Explicit movie rating':'明确评分','No rating':'未评分','Explicitly liked':'明确喜欢','Explicitly watched':'明确看过','Save explicit input':'保存手动记录',
  'Enter the exact title column from your CSV header:':'请输入 CSV 表头中影片名称列的准确名称：','{count} known watched films · profile version {version}':'已知看过 {count} 部 · 资料版本 {version}',
  'Room ready. Invite friends, then set tonight’s limits.':'房间已创建。邀请朋友后，设置今晚的条件。','Limits updated. Previous candidate responses were cleared if the inputs changed.':'条件已保存。若关键输入变更，原候选表态已清除。','Rechecking supply…':'正在重新核对供应…','Choice saved.':'选择已保存。','Invite copied.':'邀请链接已复制。','Select and copy the invitation link.':'请选中并复制邀请链接。',
  'End this room and clear personal session copies? Reusable profiles remain.':'结束房间并清理当晚个人资料副本？长期个人资料将继续保留。','Local model is working via MCP (2025-11-25). You can keep using the forms.':'本地模型正在通过 MCP（2025-11-25）处理，你仍可使用页面表单。','MCP tool trace:':'MCP 工具执行记录：','executed':'已执行','refused':'已拒绝','unreported':'未报告','Status:':'状态：','Model:':'模型：','Local assistant unavailable. {reason} Use the forms to continue.':'本地助手暂不可用。{reason} 请通过页面继续操作。',
  'Save the recovery code somewhere private. It controls this profile and cannot be recovered by email.':'请私密保存恢复码。它用于管理这份资料，无法通过邮件找回。','Stable preferences saved. Existing room copies did not change.':'长期偏好已保存，已有房间副本保持原状。','Delete this profile and withdraw linked data from active rooms?':'删除这份资料，并撤回活动房间中的关联数据？','Profile deleted and linked active copies withdrawn.':'资料已删除，活动房间中的关联副本已撤回。','Ranking changed. Find films to refresh the order.':'排序方式已更改，请重新查找电影。',
  'Preferences saved. Updated limits need a new shortlist.':'偏好已保存，请按新条件重新查找候选。','Profile copied into tonight’s record.':'资料已复制到今晚的成员记录。','Explicit preferences updated.':'手动偏好已更新。','Import complete. Review matched and pending titles; watched is not a like.':'导入完成。请查看匹配及待确认影片；已看记录不作为喜欢的证据。',
  'Secure random generation is unavailable in this browser.':'此浏览器无法生成安全随机数。','The room changed. Open the member again.':'房间已变更，请重新打开成员编辑。','The room changed.':'房间已变更。','The proposed film is outside the current library. Find films again.':'提议影片不在当前片库中，请重新查找。','Could not load the proposed film’s details. {reason}':'无法加载提议影片的资料。{reason}','The room’s inputs changed during the search. Find films again with the new limits.':'查询期间房间条件已变更，请按新条件重新查找。','Choose a film.':'请选择电影。','Enter an explicit IMDb movie ID.':'请输入准确的 IMDb 电影编号。','Choose a CSV file.':'请选择 CSV 文件。','Enter the full profile-id.secret recovery code.':'请输入完整的 profile-id.secret 格式恢复码。',
  'The room changed before this edit was saved. Review the current limits, or reopen the member editor, then try again.':'保存前房间已变更。请查看最新条件，或重新打开成员编辑后重试。','The room changed while you were editing. Submit your edit to check its version, or refresh to review.':'编辑期间房间已变更。提交时会检查版本，也可刷新后重新查看。','This room has ended. Personal session copies are no longer available.':'房间已结束，当晚个人资料副本已清理。',
  '{count}-film':'{count} 部','{count} movies in this bounded catalog. {availability}. Local model: {model}.':'当前片库含 {count} 部电影。供应：{availability}。本地模型：{model}。','configured; live facts checked per movie':'已配置，逐片查询实时供应','unconfigured':'未配置','external adapter configured':'已配置外部适配器','installed; tool performance requires separate evidence':'模型已安装','unavailable':'不可用'
 };
 Object.assign(zh, {
  'Comedy, Drama':'喜剧、剧情（可点选下方类型）','Musical':'歌舞',
  'completed':'已完成','clarification_required':'需要澄清','unverified_reply':'未验证的回复','tool_rejected':'工具请求被拒绝','cancelled':'已取消','already_processed':'已处理',
  'unknown genre; use labels shown in the catalog':'未知电影类型，请使用片库显示的类型名称或下方选项'
 });
 const extraGenres={
  'Magic Realist':'魔幻现实','Apocalyptic':'末日','American Football':'美式橄榄球','Association Football':'足球','B Movie':'B 级片','Boxing':'拳击','Buddy':'伙伴','Buddy Cop':'搭档警探','Chase':'追逐',"Children'S":'儿童','Christmas':'圣诞','Coming-of-Age':'成长','Crossover':'跨类型','Cyberpunk':'赛博朋克','Detective':'侦探','Dystopian':'反乌托邦','Epic':'史诗','Erotic':'情色','Female Buddy':'女性伙伴','Film-Noir':'黑色电影','Flashback':'倒叙','Gangster':'黑帮','Ghost':'鬼怪','Girls With Guns':'持枪女主','Heist':'劫案','Historical':'历史题材','Hood':'街区','Independent':'独立电影','Knight':'骑士','Lgbtq-Related':'LGBTQ 题材','Martial Arts':'武侠与功夫','Medieval':'中世纪','Monster':'怪兽','Parody':'恶搞','Pirate':'海盗','Police Procedural':'刑侦','Post-Apocalyptic':'末日后','Prison':'监狱','Pritcha':'寓言','Rape And Revenge':'性暴力与复仇','Road Movie':'公路电影','Satire':'讽刺','Sexploitation':'性剥削题材','Slasher':'砍杀','Speculative Fiction':'推想虚构','Sport':'体育','Spy':'间谍','Superhero':'超级英雄','Supernatural':'超自然','Survival':'生存','Suspense':'悬念','Swashbuckler':'侠盗冒险','Sword-And-Sandal':'古典史诗','Teen':'青少年','Time-Travel':'时间旅行','Transgender':'跨性别','Treasure Hunt':'寻宝','Trial':'法庭','Vampire':'吸血鬼','Vigilante':'义警'
 };
 Object.assign(zh,extraGenres,{
  'Waiting for {count} more acceptance record.':'还需 {count} 条接受记录。',
  'current subscription supply is unknown; no choice saved':'当前订阅供应无法确认，选择尚未保存。'
 });
 Object.assign(zh,{
  "Try a longer runtime": "试算片长",
  "247-film library · Data & sources ↗": "247 部片库 · 数据与来源 ↗",
  "EXPLORE BEFORE YOU CHANGE": "先试算，再决定",
  "Only the runtime changes in this preview. Your saved room stays as it is.": "仅试算片长变化，不修改已保存的房间条件。",
  "What if the limit were…": "假设片长上限为…",
  "Preview only": "只读试算",
  "Close runtime preview": "关闭片长试算",
  "Alexa+ Web Simulation · MCP Streamable HTTP 2025-11-25 · 8 tools": "Alexa+ 网页模拟 · MCP Streamable HTTP 2025-11-25 · 8 项工具",
  "What if we had 15 more minutes? ↗": "如果多给 15 分钟？↗",
  "What if we had 15 more minutes? Preview without changing any limits.": "如果多给 15 分钟会怎样？请只做试算，不修改任何条件。",
  "Assistant mode": "助手模式",
  "Change or recommend": "修改条件或推荐",
  "Read-only runtime preview": "只读片长试算",
  "Optional: pick 1–3 films they explicitly like": "可跳过：选 1–3 部明确喜欢的电影",
  "Skip this if you prefer. Watched films are not treated as likes.": "可以跳过。看过不代表喜欢。",
  "Search liked films": "搜索喜欢的电影",
  "Remove": "移除",
  "{count}/3 selected": "已选 {count}/3 部",
  "Choose at most three films here. More explicit inputs remain available under Watched films & ratings.": "这里最多选择 3 部。更多明确输入可在“已看电影与评分”中填写。",
  "No matching film in this bounded library.": "当前有限片库中未找到匹配电影。",
  "Hard conditions matched · no candidate-specific preference evidence": "满足硬性条件 · 暂无该候选的偏好依据",
  "Some explicit preference evidence": "部分成员有明确偏好依据",
  "Explicit preference evidence for each member": "各成员均有明确偏好依据",
  "Ranking has little preference evidence; the catalog baseline is not a predicted personal rating.": "偏好排序依据不足；片库基线不代表个人评分预测。",
  "READ-ONLY WHAT-IF": "只读假设试算",
  "{before} → {after} minutes": "{before} → {after} 分钟",
  "{before} → {after} eligible films · {added} newly eligible": "可行电影 {before} → {after} 部 · 实际新增 {added} 部",
  "{count} films have unknown metadata or supply under this limit. They are excluded.": "拟定片长下有 {count} 部电影资料或供应未知，已排除。",
  "Cached supply only. Platforms, vetoes and rewatch rules stay unchanged. Find films can refresh missing supply.": "仅使用当前有效供应缓存。平台、类型否决与重看规则不变。“查找电影”可补查供应。",
  "The room or cached supply changed. Run this preview again.": "房间或供应缓存已变化，请重新试算。",
  "Apply this runtime": "应用这个片长",
  "Runtime applied. Previous candidates and responses were cleared. Find films with the new saved limit.": "已应用片长，旧候选与表态已清除。请按新条件查找电影。",
  "Runtime applied. Find films to use the new limit.": "片长已应用，请按新条件查找电影。",
  "No additional verified film qualifies. Review the other limits or refresh supply.": "没有新增可核验的电影。请检查其他条件或补查供应。",
  "preview inputs or cached facts changed; run the preview again": "试算条件或缓存事实已变化，请重新试算。",
  "no additional verified film qualifies; review the saved limits instead": "没有新增可核验的电影，请检查已保存的条件。",
  "preview runtime must exceed the saved limit and be at most 600 minutes": "试算片长必须大于已保存上限，且不超过 600 分钟。",
  "preview_stale": "试算已过时",
  "Apocalyptic": "末日"
});
 const bindings = new Map();
 let language;
 try { language = localStorage.getItem('commonreel.language'); } catch {}
 if (!['en','zh-CN'].includes(language)) language = (navigator.language || 'en').toLowerCase().startsWith('zh') ? 'zh-CN' : 'en';
 function t(source, values = {}) {
  const template = language === 'zh-CN' ? zh[source] || source : source;
  const args = typeof values === 'function' ? values() : values;
  return template.replace(/\{(\w+)\}/g, (match,key) => Object.hasOwn(args,key) ? String(args[key]) : match);
 }
 function draw(target,binding) {
  const value = binding.prefix + t(binding.source,binding.values) + binding.suffix;
  if(binding.attribute) target.setAttribute(binding.attribute,value); else target.textContent=value;
 }
 function bind(target, source, values = {}, attribute = '', prefix = '', suffix = '') {
  let records=bindings.get(target);if(!records){records=new Map();bindings.set(target,records);}
  const binding={source,values,attribute,prefix,suffix};records.set(attribute,binding);draw(target,binding);return target;
 }
 function text(node,source,values={}) { node.textContent='';bind(node.appendChild(document.createTextNode('')),source,values);return node; }
 function node(source,values={}) { return bind(document.createTextNode(''),source,values); }
 function attribute(target,name,source,values={}) { bind(target,source,values,name); }
 function staticUI(root) {
  const walker=document.createTreeWalker(root,NodeFilter.SHOW_TEXT);const texts=[];
  while(walker.nextNode())texts.push(walker.currentNode);
  for(const target of texts){const source=target.textContent.trim();if(Object.hasOwn(zh,source))bind(target,source,{},'',target.textContent.match(/^\s*/)[0],target.textContent.match(/\s*$/)[0]);}
  for(const element of root.querySelectorAll('[aria-label],[placeholder],[title],[data-prompt]'))for(const name of ['aria-label','placeholder','title','data-prompt']){const value=element.getAttribute(name);if(Object.hasOwn(zh,value))attribute(element,name,value);}
 }
 function prune() { for(const target of bindings.keys())if(!target.isConnected)bindings.delete(target); }
 function apply() {
  document.documentElement.lang=language;document.title=t('CommonReel · Good films. Better company.');
  for(const [target,records] of bindings){if(!target.isConnected){bindings.delete(target);continue;}for(const binding of records.values())draw(target,binding);}
  for(const select of document.querySelectorAll('[data-language]'))select.value=language;
 }
 function setLanguage(next) {
  if(!['en','zh-CN'].includes(next))return;
  language=next;try{localStorage.setItem('commonreel.language',next);}catch{}
  apply();document.dispatchEvent(new Event('languagechange'));
 }
 const genres=['Comedy','Drama','Action','Thriller','Horror','Romance','Sci-Fi','Animation','Adventure','Crime','Mystery','Fantasy','Family','Documentary','Music','War','Western','History','Biography','Musical',...Object.keys(extraGenres)];
 function canonicalGenre(value) { return genres.find(genre=>zh[genre]===value)||value; }
 return {t,text,node,attribute,staticUI,apply,prune,setLanguage,canonicalGenre,get language(){return language;},messages:zh};
})();
