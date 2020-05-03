import traceback

#
# Routing
#

# 2019-11-21T02:13:58+00:00
#   Will be deprecated, use "acl" instead
PublicMode = 1
TraceMode = 2
ProtectMode = 3

class ACL:
    PublicMode = 1
    TraceMode = 2
    ProtectMode = 3

# cObjshRouter is an ObjshRouter instance to objsh.Router(golang). 
# (cObjshRouter is defined in iap_patched.c)
cObjshRouter = None



import importlib, sys
class ReloadableRouterWrapper(object):
    def __init__(self):
        self.handlers = {}

    def reloadModule(self,name):
        try:
            importlib.reload(sys.modules[name])
            return 'reload module "' + name +'" completed' 
        except:
            return traceback.format_exc()

    def register(self,method,path,acl):
        def f(handler):
            try:
                self.handlers[path]
            except KeyError:
                # register this path at frist time
                def registeredHandler(*args,**kw):
                    return self.handlers[path](*args,**kw)
                if method == 'Get':
                    cObjshRouter.Get(path,registeredHandler,acl)
                elif method == 'Post':
                    cObjshRouter.Post(path,registeredHandler,acl)
                elif method == 'Websocket':
                    cObjshRouter.Websocket(path,registeredHandler,acl)
                elif method == 'FileUpload':
                    cObjshRouter.FileUpload(path,registeredHandler,acl)
                else:
                    raise NotImplementedError(method+' not implmented')
            self.handlers[path] = handler
        return f

class RouterWrapper(object):
    def __init__(self):
        self.reloadableRouter = ReloadableRouterWrapper()

    def reloadModule(self,name):
        return  self.reloadableRouter.reloadModule(name)

    def Get(self,path,acl,reloadable=False):	
        if reloadable:
            return self.reloadableRouter.register('Get',path,acl)
        else:
            def f(handler):
                cObjshRouter.Get(path,handler,acl)
            return f

    def Post(self,path,acl,reloadable=False):	
        if reloadable:
            return self.reloadableRouter.register('Post',path,acl)
        else:
            def f(handler):
                cObjshRouter.Post(path,handler,acl)
            return f

    def Websocket(self,path,acl,reloadable=False):
        if reloadable:
            return self.reloadableRouter.register('Websocket',path,acl)
        else:
            def f(handler):
                cObjshRouter.Websocket(path,handler,acl)
            return f

    def FileUpload(self,path,acl,reloadable=False):
        if reloadable:
            return self.reloadableRouter.register('FileUpload',path,acl)
        else:
            def f(handler):
                cObjshRouter.FileUpload(path,handler,acl)
            return f
    # Not implemented yet
    #def Wsgi(self,path,acl,reloadable=False):	
    #	def f(handler):
    #		cObjshRouter.Wsgi(path,handler,acl)
    #	return f

Router = RouterWrapper()
#ReloadableRouter = ReloadableRouterWrapper()

#
# Tree API
#

# cObjshTree is an ObjshTree instance to objsh.Tree(golang). 
# (cObjshTree is defined in iap_patched.c)
cObjshTree = None

class BaseBranch(object):
    def __init__(self,name=None):
        self.name = name
        self.exportableNames = []
        self.exportableDocs = {}
    def getExportableNames(self):
        return self.exportableNames[:]
    #def beReady(self,tree):
    #    # return False if root.SureReady is going to be called later manually
    #    return True
    def beReady(self,tree):
        raise NotImplementError('%s.beReady not implemented' % self)
    def _beReady(self,treeName):
        try:
            return self.beReady(getattr(Tree,treeName))
        except:
            traceback.print_exc()
            raise
    def __call__(self,methodName,ctx):
        if not methodName in self.exportableNames:
            raise AttributeError(methodName + ' is not exported')
        try:
            getattr(self,methodName)(ctx)
        except:
            ctx.reject(400,traceback.format_exc()) 
    def export(self,*funcs):
        self.exportableNames = []
        self.exportableDocs = {}
        for func in funcs:
            print("exporting",func.__name__)
            self.exportableNames.append(func.__name__)
            self.exportableDocs[func.__name__] = func.__doc__

class PesudoTree(object):
    def __init__(self,name):
        self.name = name
    def addBranch(self,branchObj,branchName=None):
        try:
            if branchName is not None:
                assert isinstance(branchName,str), 'addBranch(branchObj,branchName) where branchName should be string'
                branchObj.name = branchName
            cObjshTree.AddBranch(branchObj,self.name)
        except:
            print(traceback.format_exc())
    addBranchWithName = addBranch
    def sureReady(self,branchObj):
        cObjshTree.SureReady(self.name, branchObj.name)

class PyTreeWrapper(object):
    # Wrap trees in golang for python scripts
    def __init__(self):
        pass

    def addTree(self,name,*args):
        # Called in golang, to make a python statement be valid, such as
        # Tree.UnitTest.addBranch, Tree.Member.addBranch
        tree = PesudoTree(name)
        setattr(self,name,tree)


initCallables = []
def callWhenRunning(func,*args,**kw):
    initCallables.append((func,args,kw))
def callInitCallables():
    for func,args,kw in initCallables:
        func(*args,**kw)

# Prefer GoTrees than Tree
GoTrees = Tree = PyTreeWrapper()
# Utility var to be called in golang
_addTree = GoTrees.addTree

__all__ = ['Router','ACL','GoTrees','Tree','BaseBranch','callWhenRunning',
           'PublicMode','TraceMode','ProtectMode']
